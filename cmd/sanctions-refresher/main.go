// cmd/sanctions-refresher — downloads the sanctions lists (OFAC SDN, UN consolidated, EU FSF, COAF) and publishes
// one validated snapshot per list for the screening pods (MS-024d 1.7, RTSK.024.1207.01).
//
// Usage:
//
//	sanctions-refresher --providers all|ofac|un|eu|coaf[,…] [--snapshot-dir DIR] [--stagger 15m] [--timeout 5m]
//
// What it produces. For each list, «<snapshot-dir>/<list>.json» (sanctions.SnapshotPath): a sanctions.Snapshot
// with the list's source, fetched_at (UTC, when the download started) and the entries the list's provider accepted.
// Each list goes through the same sanctions.ListProvider the screening path uses (fetch → parse → validate), into
// an index that only records the entries; the snapshot is then written with sanctions.WriteSnapshot (temp file in
// the same directory, fsync, rename), so a reader sees the previous snapshot or the new one, never a partial file.
// A list that fails (download, parse, an invalid or foreign entry, an empty list) leaves its previous snapshot
// untouched, the other lists are still refreshed, and the run exits non-zero.
//
// Who consumes it. The processes that call Match (the API, through compliance screening) mount the same volume
// read-only, sanctions.ReadSnapshot each list and Load it into their own sanctions.Cache. The hand-off is a file
// because this binary runs as a CronJob (MS-024d 1.10): a Job cannot fill another pod's memory.
//
// Configuration (12-factor, env):
//
//	EXCHANGEOS_SANCTIONS_<LIST>_URL     download URL of each selected list (https only); LIST is OFAC, UN, EU, COAF.
//	                                    It may carry a credential (the EU FSF token), so it is never logged.
//	EXCHANGEOS_SANCTIONS_SNAPSHOT_DIR   default for --snapshot-dir (/var/lib/exchangeos/sanctions)
//	EXCHANGEOS_ENV                      logger format (internal/config)
//
// Freshness (MS-024d 1.8): after each list the run records sanctions_list_age_seconds{list}, the age of the snapshot
// the list has on disk, and exports it over OTLP (internal/telemetry) before exiting. See freshness.go.
//
// --stagger spaces the lists inside one run, so the upstreams are not hit at the same instant. Only public list
// data passes through here; no screening query (counterparty name) ever does.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/internal/config"
	"github.com/revenutech/exchangeos/internal/telemetry"
	"github.com/revenutech/exchangeos/pkg/sanctions"
	"github.com/revenutech/exchangeos/pkg/sanctions/coaf"
	"github.com/revenutech/exchangeos/pkg/sanctions/eu"
	"github.com/revenutech/exchangeos/pkg/sanctions/ofac"
	"github.com/revenutech/exchangeos/pkg/sanctions/un"
)

const (
	serviceName        = "exchangeos-sanctions-refresher"
	serviceVersion     = "0.1.0-dev"
	defaultSnapDir     = "/var/lib/exchangeos/sanctions"
	defaultMaxBytes    = 256 << 20 // the providers' own caps are of the same order (ofac/un MaxListBytes)
	defaultListTimeout = 5 * time.Minute
)

// errTooLarge is returned while reading a download that exceeds --max-bytes. The body is never truncated: a cut
// CSV would still parse, and load half a list.
var errTooLarge = errors.New("sanctions-refresher: download exceeds --max-bytes")

// target is one list the refresher knows how to build.
type target struct {
	name        string // flag and file name: ofac, un, eu, coaf
	source      sanctions.Source
	newProvider func(sanctions.Fetch, sanctions.Index) (*sanctions.ListProvider, error)
}

// targets is every list, in the order «all» refreshes them.
var targets = []target{
	{name: "ofac", source: sanctions.SourceOFAC, newProvider: ofac.New},
	{name: "un", source: sanctions.SourceUN, newProvider: un.New},
	{name: "eu", source: sanctions.SourceEU, newProvider: eu.New},
	{name: "coaf", source: sanctions.SourceCOAF, newProvider: coaf.New},
}

// job is a target with the Fetch that downloads it.
type job struct {
	target
	fetch sanctions.Fetch
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("sanctions-refresher", flag.ContinueOnError)
	providers := fs.String("providers", "all", "lists to refresh: all, or a comma-separated subset of ofac,un,eu,coaf")
	dir := fs.String("snapshot-dir", envOr("EXCHANGEOS_SANCTIONS_SNAPSHOT_DIR", defaultSnapDir), "directory of the per-list snapshot files")
	stagger := fs.Duration("stagger", 0, "wait between two lists of the same run")
	timeout := fs.Duration("timeout", defaultListTimeout, "deadline of one list (download, parse, write)")
	maxBytes := fs.Int64("max-bytes", defaultMaxBytes, "largest download accepted per list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	selected, err := parseProviders(*providers)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger, err := telemetry.NewLogger(cfg.Env)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	jobs, err := buildJobs(selected, os.Getenv, newHTTPClient(*timeout), *maxBytes)
	if err != nil {
		return err
	}
	logger.Info("sanctions-refresher starting",
		zap.String("env", cfg.Env),
		zap.String("version", serviceVersion),
		zap.String("providers", *providers),
		zap.Duration("stagger", *stagger),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdownTelemetry, err := telemetry.Init(ctx, telemetry.Options{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		OTLPEndpoint:   cfg.OTel.Endpoint,
		Env:            cfg.Env,
	})
	if err != nil {
		return fmt.Errorf("telemetry init: %w", err)
	}
	defer func() {
		// A Job exits right after the run: the shutdown is what exports the age gauge.
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Warn("telemetry shutdown", zap.Error(err))
		}
	}()
	age, err := newAgeGauge(otel.Meter("github.com/revenutech/exchangeos/cmd/sanctions-refresher"))
	if err != nil {
		return err
	}

	r := &refresher{dir: *dir, stagger: *stagger, timeout: *timeout, now: time.Now, log: logger, age: age}
	return r.run(ctx, jobs)
}

// parseProviders turns --providers into targets: «all» alone, or distinct list names separated by commas.
func parseProviders(s string) ([]target, error) {
	if strings.TrimSpace(s) == "all" {
		return append([]target(nil), targets...), nil
	}
	var out []target
	seen := map[string]bool{}
	for _, raw := range strings.Split(s, ",") {
		name := strings.ToLower(strings.TrimSpace(raw))
		if seen[name] {
			continue
		}
		var found bool
		for _, t := range targets {
			if t.name == name {
				out, found = append(out, t), true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("--providers: unknown list %q (want all, or a subset of ofac,un,eu,coaf)", name)
		}
		seen[name] = true
	}
	return out, nil
}

func urlEnv(t target) string { return "EXCHANGEOS_SANCTIONS_" + strings.ToUpper(t.name) + "_URL" }

// buildJobs reads the download URL of each target. The URL may embed a credential, so no error quotes it.
func buildJobs(ts []target, getenv func(string) string, client *http.Client, maxBytes int64) ([]job, error) {
	jobs := make([]job, 0, len(ts))
	for _, t := range ts {
		key := urlEnv(t)
		raw := strings.TrimSpace(getenv(key))
		if raw == "" {
			return nil, fmt.Errorf("%s is required to refresh %s", key, t.name)
		}
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid URL", key)
		}
		if u.Scheme != "https" || u.Host == "" {
			return nil, fmt.Errorf("%s must be an https URL", key)
		}
		jobs = append(jobs, job{target: t, fetch: httpFetch(client, t.name, raw, maxBytes)})
	}
	return jobs, nil
}

// maxRedirects bounds the redirects one download may follow.
const maxRedirects = 5

var (
	// errInsecureRedirect refuses a redirect off https: the list would come in clear, outside TLS 1.3, and could be
	// poisoned on the way.
	errInsecureRedirect = errors.New("sanctions-refresher: redirect to a non-https URL refused")
	// errTooManyRedirects stops a redirect chain longer than maxRedirects.
	errTooManyRedirects = errors.New("sanctions-refresher: too many redirects")
)

// newHTTPClient is the client of the list downloads: TLS 1.3 minimum (platform rule), redirects only to https and
// at most maxRedirects of them, one deadline per request.
func newHTTPClient(timeout time.Duration) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS13}
	return &http.Client{Timeout: timeout, Transport: tr, CheckRedirect: checkRedirect}
}

// checkRedirect is the redirect policy of newHTTPClient. Its errors never quote a URL: the target (Location) may carry
// the same token as the original. The client wraps them in a *url.Error, which httpFetch strips.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return errInsecureRedirect
	}
	if len(via) >= maxRedirects {
		return errTooManyRedirects
	}
	return nil
}

// stripURL drops the URL from a *url.Error (which quotes it, query string included) and keeps the cause.
func stripURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s: %w", ue.Op, ue.Err)
	}
	return err
}

// httpFetch downloads rawURL, refusing a non-200 answer and a body over maxBytes. Errors name the list, never the URL.
func httpFetch(client *http.Client, name, rawURL string, maxBytes int64) sanctions.Fetch {
	return func(ctx context.Context) (io.ReadCloser, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: build request: %w", name, stripURL(err))
		}
		req.Header.Set("User-Agent", serviceName+"/"+serviceVersion)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%s: download: %w", name, stripURL(err))
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("%s: download: HTTP %d", name, resp.StatusCode)
		}
		return &cappedBody{r: io.LimitReader(resp.Body, maxBytes+1), c: resp.Body, max: maxBytes}, nil
	}
}

// cappedBody fails the read, instead of truncating, once more than max bytes arrived.
type cappedBody struct {
	r      io.Reader
	c      io.Closer
	n, max int64
}

func (b *cappedBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.n += int64(n)
	if b.n > b.max {
		return 0, errTooLarge
	}
	return n, err
}

func (b *cappedBody) Close() error { return b.c.Close() }

// recorder is the Index the refresher hands to a ListProvider: it keeps what Refresh accepted, to be written as the
// snapshot. It never matches; the screening pods do, from the snapshot.
type recorder struct{ entries []sanctions.Entry }

func (r *recorder) Load(entries []sanctions.Entry) error {
	r.entries = entries
	return nil
}

func (r *recorder) Match(sanctions.MatchQuery) ([]sanctions.Match, error) {
	return nil, errors.New("sanctions-refresher: the recorder index does not match")
}

type refresher struct {
	dir     string
	stagger time.Duration
	timeout time.Duration // per list; zero means only the parent context bounds it
	now     func() time.Time
	log     *zap.Logger
	age     *ageGauge // sanctions_list_age_seconds; nil records nothing
}

// outcome is the result of one list's refresh.
type outcome struct {
	entries   int
	fetchedAt time.Time
	duration  time.Duration
	err       error
	// age is how old the list's snapshot on disk is once the refresh ended: the one just written, or the previous
	// one when the refresh failed. ageKnown is false when the list has no readable snapshot at all.
	age      time.Duration
	ageKnown bool
}

// run refreshes the jobs in order, waiting stagger between two of them. A failed list does not stop the others; the
// returned error joins every failure, and the context's error when the run was interrupted.
func (r *refresher) run(ctx context.Context, jobs []job) error {
	var errs []error
	for i, j := range jobs {
		if i > 0 && r.stagger > 0 {
			if err := sleep(ctx, r.stagger); err != nil {
				return errors.Join(append(errs, err)...)
			}
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		if out := r.refresh(ctx, j); out.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", j.name, out.err))
		}
	}
	return errors.Join(errs...)
}

// refresh runs one list through its ListProvider and, when it is accepted, publishes the snapshot.
func (r *refresher) refresh(ctx context.Context, j job) outcome {
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}
	start := r.now()
	out := outcome{fetchedAt: start.UTC()}
	out.err = r.publish(ctx, j, out.fetchedAt, &out.entries)
	end := r.now()
	out.duration = end.Sub(start)
	r.measureAge(ctx, j, end, &out)
	if out.err != nil {
		fields := []zap.Field{zap.String("list", j.name), zap.Duration("duration", out.duration), zap.Error(out.err)}
		if out.ageKnown {
			fields = append(fields, zap.Duration("snapshot_age", out.age))
		}
		r.log.Error("list refresh failed", fields...)
		return out
	}
	r.log.Info("list refreshed",
		zap.String("list", j.name), zap.Int("entries", out.entries), zap.Duration("duration", out.duration),
		zap.Time("fetched_at", out.fetchedAt))
	return out
}

// measureAge sets out.age from the snapshot the list now has on disk and reports it on the age gauge. A failed
// refresh left the previous snapshot in place, so its age keeps growing run after run until the SLO alert fires.
func (r *refresher) measureAge(ctx context.Context, j job, now time.Time, out *outcome) {
	switch {
	case out.err == nil:
		out.age, out.ageKnown = now.Sub(out.fetchedAt), true
	default:
		snap, err := sanctions.ReadSnapshot(r.dir, j.source)
		if err != nil {
			r.log.Warn("no snapshot to report the age of", zap.String("list", j.name), zap.Error(err))
			return
		}
		out.age, out.ageKnown = now.Sub(snap.FetchedAt), true
	}
	r.age.record(context.WithoutCancel(ctx), j.name, out.age)
}

func (r *refresher) publish(ctx context.Context, j job, fetchedAt time.Time, entries *int) error {
	rec := &recorder{}
	p, err := j.newProvider(j.fetch, rec)
	if err != nil {
		return err
	}
	if err := p.Refresh(ctx); err != nil {
		return err
	}
	if err := sanctions.WriteSnapshot(r.dir, sanctions.Snapshot{Source: j.source, FetchedAt: fetchedAt, Entries: rec.entries}); err != nil {
		return err
	}
	*entries = len(rec.entries)
	return nil
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

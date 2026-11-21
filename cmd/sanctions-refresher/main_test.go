package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/revenutech/exchangeos/pkg/sanctions"
	"github.com/revenutech/exchangeos/pkg/sanctions/coaf"
	"github.com/revenutech/exchangeos/pkg/sanctions/eu"
	"github.com/revenutech/exchangeos/pkg/sanctions/ofac"
	"github.com/revenutech/exchangeos/pkg/sanctions/un"
)

var testNow = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

// fixtures are the synthetic lists already committed with each provider.
var fixtures = map[string]string{
	"ofac": filepath.Join("..", "..", "pkg", "sanctions", "ofac", "testdata", "sdn.xml"),
	"un":   filepath.Join("..", "..", "pkg", "sanctions", "un", "testdata", "consolidated.xml"),
	"eu":   filepath.Join("..", "..", "pkg", "sanctions", "eu", "testdata", "eu-fsf-sintetico.csv"),
	"coaf": filepath.Join("..", "..", "pkg", "sanctions", "coaf", "testdata", "coaf-sintetico.xlsx"),
}

var parsers = map[string]sanctions.Parse{"ofac": ofac.Parse, "un": un.Parse, "eu": eu.Parse, "coaf": coaf.Parse}

func fileFetch(path string) sanctions.Fetch {
	return func(context.Context) (io.ReadCloser, error) { return os.Open(path) }
}

func failingFetch(err error) sanctions.Fetch {
	return func(context.Context) (io.ReadCloser, error) { return nil, err }
}

func mustTarget(t *testing.T, name string) target {
	t.Helper()
	ts, err := parseProviders(name)
	require.NoError(t, err)
	require.Len(t, ts, 1)
	return ts[0]
}

func newTestRefresher(dir string) (*refresher, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return &refresher{dir: dir, now: func() time.Time { return testNow }, log: zap.New(core)}, logs
}

func TestParseProviders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{in: "all", want: []string{"ofac", "un", "eu", "coaf"}},
		{in: "ofac", want: []string{"ofac"}},
		{in: "un,coaf", want: []string{"un", "coaf"}},
		{in: " EU , ofac ", want: []string{"eu", "ofac"}},
		{in: "ofac,ofac", want: []string{"ofac"}},
		{in: "", wantErr: true},
		{in: "ofac,", wantErr: true},
		{in: "sdn", wantErr: true},
		{in: "all,ofac", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got, err := parseProviders(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			names := make([]string, 0, len(got))
			for _, tg := range got {
				names = append(names, tg.name)
			}
			assert.Equal(t, tt.want, names)
		})
	}
}

func TestBuildJobs(t *testing.T) {
	t.Parallel()
	const secret = "s3cr3t-token"
	all, err := parseProviders("all")
	require.NoError(t, err)
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{name: "every url set", env: map[string]string{
			"EXCHANGEOS_SANCTIONS_OFAC_URL": "https://ofac.example.test/sdn.xml",
			"EXCHANGEOS_SANCTIONS_UN_URL":   "https://un.example.test/consolidated.xml",
			"EXCHANGEOS_SANCTIONS_EU_URL":   "https://eu.example.test/fsf.csv?token=" + secret,
			"EXCHANGEOS_SANCTIONS_COAF_URL": "https://coaf.example.test/lista.xlsx",
		}},
		{name: "missing url", env: map[string]string{
			"EXCHANGEOS_SANCTIONS_OFAC_URL": "https://ofac.example.test/sdn.xml",
			"EXCHANGEOS_SANCTIONS_UN_URL":   "https://un.example.test/consolidated.xml",
			"EXCHANGEOS_SANCTIONS_COAF_URL": "https://coaf.example.test/lista.xlsx",
		}, wantErr: "EXCHANGEOS_SANCTIONS_EU_URL"},
		{name: "plain http refused", env: map[string]string{
			"EXCHANGEOS_SANCTIONS_OFAC_URL": "https://ofac.example.test/sdn.xml",
			"EXCHANGEOS_SANCTIONS_UN_URL":   "https://un.example.test/consolidated.xml",
			"EXCHANGEOS_SANCTIONS_EU_URL":   "http://eu.example.test/fsf.csv?token=" + secret,
			"EXCHANGEOS_SANCTIONS_COAF_URL": "https://coaf.example.test/lista.xlsx",
		}, wantErr: "EXCHANGEOS_SANCTIONS_EU_URL"},
		{name: "unparseable url", env: map[string]string{
			"EXCHANGEOS_SANCTIONS_OFAC_URL": "https://ofac.example.test/sdn.xml",
			"EXCHANGEOS_SANCTIONS_UN_URL":   "https://un.example.test/consolidated.xml",
			"EXCHANGEOS_SANCTIONS_EU_URL":   "https://eu.example.test/%zz?token=" + secret,
			"EXCHANGEOS_SANCTIONS_COAF_URL": "https://coaf.example.test/lista.xlsx",
		}, wantErr: "EXCHANGEOS_SANCTIONS_EU_URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(k string) string { return tt.env[k] }
			jobs, err := buildJobs(all, getenv, http.DefaultClient, 1<<20)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.NotContains(t, err.Error(), secret, "the url (and its token) must not reach the error")
				return
			}
			require.NoError(t, err)
			require.Len(t, jobs, 4)
			for i, j := range jobs {
				assert.Equal(t, all[i].name, j.name)
				assert.NotNil(t, j.fetch)
			}
		})
	}
}

func TestNewHTTPClient_TLS13(t *testing.T) {
	t.Parallel()
	c := newHTTPClient(time.Minute)
	assert.Equal(t, time.Minute, c.Timeout)
	tr, ok := c.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, tr.TLSClientConfig)
	assert.Equal(t, uint16(tls.VersionTLS13), tr.TLSClientConfig.MinVersion)
}

func TestHTTPFetch(t *testing.T) {
	t.Parallel()
	const secret = "s3cr3t-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = io.WriteString(w, "0123456789")
		case "/big":
			_, _ = io.WriteString(w, strings.Repeat("x", 64))
		default:
			http.Error(w, "down", http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(srv.Close)
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	tests := []struct {
		name     string
		url      string
		maxBytes int64
		want     string
		openErr  string // substring of the error returned by fetch
		readErr  error  // error returned while reading the body
	}{
		{name: "ok", url: srv.URL + "/ok?token=" + secret, maxBytes: 10, want: "0123456789"},
		{name: "http error", url: srv.URL + "/down?token=" + secret, maxBytes: 10, openErr: "503"},
		{name: "over the cap", url: srv.URL + "/big?token=" + secret, maxBytes: 10, readErr: errTooLarge},
		{name: "unreachable", url: closedURL + "/ok?token=" + secret, maxBytes: 10, openErr: "ofac"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fetch := httpFetch(srv.Client(), "ofac", tt.url, tt.maxBytes)
			rc, err := fetch(context.Background())
			if tt.openErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.openErr)
				assert.NotContains(t, err.Error(), secret, "the url (and its token) must not reach the error")
				return
			}
			require.NoError(t, err)
			defer func() { _ = rc.Close() }()
			b, err := io.ReadAll(rc)
			if tt.readErr != nil {
				require.ErrorIs(t, err, tt.readErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}

// Every provider refreshes from its committed fixture into a snapshot that holds exactly what the parser produced.
func TestRefresh_WritesSnapshotPerList(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"ofac", "un", "eu", "coaf"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			raw, err := os.Open(fixtures[name])
			require.NoError(t, err)
			want, err := parsers[name](raw)
			_ = raw.Close()
			require.NoError(t, err)

			dir := t.TempDir()
			r, logs := newTestRefresher(dir)
			tg := mustTarget(t, name)
			out := r.refresh(context.Background(), job{target: tg, fetch: fileFetch(fixtures[name])})
			require.NoError(t, out.err)
			assert.Equal(t, len(want), out.entries)

			snap, err := sanctions.ReadSnapshot(dir, tg.source)
			require.NoError(t, err)
			assert.Equal(t, tg.source, snap.Source)
			assert.True(t, testNow.Equal(snap.FetchedAt))
			assert.Equal(t, want, snap.Entries)

			done := logs.FilterMessage("list refreshed").All()
			require.Len(t, done, 1)
			fields := done[0].ContextMap()
			assert.Equal(t, name, fields["list"])
			assert.Contains(t, fields, "duration")
			assert.EqualValues(t, len(want), fields["entries"])
		})
	}
}

// A failed refresh never touches the snapshot already on disk.
func TestRefresh_FailureKeepsPreviousSnapshot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, logs := newTestRefresher(dir)
	tg := mustTarget(t, "ofac")
	require.NoError(t, r.refresh(context.Background(), job{target: tg, fetch: fileFetch(fixtures["ofac"])}).err)
	before, err := os.ReadFile(sanctions.SnapshotPath(dir, tg.source))
	require.NoError(t, err)

	tests := []struct {
		name  string
		fetch sanctions.Fetch
	}{
		{"fetch fails", failingFetch(errors.New("connection refused"))},
		{"wrong format", func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("<html>maintenance</html>")), nil
		}},
		{"other list served", fileFetch(fixtures["un"])},
	}
	for _, tt := range tests {
		r.now = func() time.Time { return testNow.Add(time.Hour) }
		out := r.refresh(context.Background(), job{target: tg, fetch: tt.fetch})
		require.Error(t, out.err, tt.name)
		after, err := os.ReadFile(sanctions.SnapshotPath(dir, tg.source))
		require.NoError(t, err)
		assert.Equal(t, before, after, tt.name)
	}
	assert.Len(t, logs.FilterMessage("list refresh failed").All(), len(tests))
}

// run refreshes every list even when one fails, and reports the failure.
func TestRun_ContinuesAfterFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, _ := newTestRefresher(dir)
	jobs := []job{
		{target: mustTarget(t, "ofac"), fetch: failingFetch(errors.New("connection refused"))},
		{target: mustTarget(t, "un"), fetch: fileFetch(fixtures["un"])},
	}
	err := r.run(context.Background(), jobs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ofac")

	_, err = sanctions.ReadSnapshot(dir, sanctions.SourceUN)
	require.NoError(t, err)
	_, err = sanctions.ReadSnapshot(dir, sanctions.SourceOFAC)
	require.Error(t, err)
}

func TestRun_AllSucceed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, logs := newTestRefresher(dir)
	var jobs []job
	for _, name := range []string{"ofac", "un", "eu", "coaf"} {
		jobs = append(jobs, job{target: mustTarget(t, name), fetch: fileFetch(fixtures[name])})
	}
	require.NoError(t, r.run(context.Background(), jobs))
	assert.Len(t, logs.FilterMessage("list refreshed").All(), 4)
	names, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, names, 4)
}

// The stagger between lists honours cancellation: a signal during the wait stops the run before the next list.
func TestRun_StaggerStopsOnCancel(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, _ := newTestRefresher(dir)
	r.stagger = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var second atomic.Bool
	jobs := []job{
		{target: mustTarget(t, "ofac"), fetch: func(c context.Context) (io.ReadCloser, error) {
			cancel() // the signal arrives while the first list is downloading
			return fileFetch(fixtures["ofac"])(c)
		}},
		{target: mustTarget(t, "un"), fetch: func(c context.Context) (io.ReadCloser, error) {
			second.Store(true)
			return fileFetch(fixtures["un"])(c)
		}},
	}
	err := r.run(ctx, jobs)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, second.Load(), "the second list must not start after cancellation")
}

func TestRefresh_TimeoutBoundsTheFetch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, _ := newTestRefresher(dir)
	r.timeout = 10 * time.Millisecond
	blocking := func(c context.Context) (io.ReadCloser, error) {
		<-c.Done()
		return nil, c.Err()
	}
	out := r.refresh(context.Background(), job{target: mustTarget(t, "ofac"), fetch: blocking})
	require.ErrorIs(t, out.err, context.DeadlineExceeded)
}

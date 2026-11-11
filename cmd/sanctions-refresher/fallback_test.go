package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

func unreachable() sanctions.Fetch {
	return failingFetch(fmt.Errorf("%w: ofac: download: dial tcp: connection refused", errUpstream))
}

func TestHTTPFetch_MarksUpstreamUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "maintenance", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	gone := httptest.NewServer(http.NotFoundHandler())
	goneURL := gone.URL
	gone.Close()

	for name, u := range map[string]string{"http 503": srv.URL, "connection refused": goneURL} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := httpFetch(srv.Client(), "ofac", u+"/sdn.xml", 1<<20)(context.Background())
			require.ErrorIs(t, err, errUpstream)
		})
	}
}

// The fallback rule, list by list: unreachable upstream with a snapshot of at most 24h serves the cached snapshot and
// is not a failure of the run; past 24h, or with no snapshot, the list is blocked; a list that answers but does not
// parse is a failure even while its cached snapshot is fresh, because that needs a person (format change, poisoning).
func TestRefresh_FallbackRule(t *testing.T) {
	t.Parallel()
	badFormat := func(context.Context) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("<html>moved</html>")), nil
	}
	tests := []struct {
		name         string
		cachedAge    time.Duration // < 0: no cached snapshot
		fetch        sanctions.Fetch
		wantFallback bool
		wantBlocked  bool
		wantRunErr   bool
	}{
		{name: "unreachable, cache 1h", cachedAge: time.Hour, fetch: unreachable(), wantFallback: true},
		{name: "unreachable, cache exactly 24h", cachedAge: freshnessSLO, fetch: unreachable(), wantFallback: true},
		{name: "unreachable, cache 24h01", cachedAge: freshnessSLO + time.Minute, fetch: unreachable(), wantBlocked: true, wantRunErr: true},
		{name: "unreachable, no cache", cachedAge: -1, fetch: unreachable(), wantBlocked: true, wantRunErr: true},
		{name: "bad format, cache 1h", cachedAge: time.Hour, fetch: badFormat, wantRunErr: true},
		{name: "bad format, cache 30h", cachedAge: 30 * time.Hour, fetch: badFormat, wantBlocked: true, wantRunErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.cachedAge >= 0 {
				require.NoError(t, writeFixtureSnapshot(dir, "ofac", testNow.Add(-tt.cachedAge)))
			}
			var before []byte
			if tt.cachedAge >= 0 {
				b, err := os.ReadFile(sanctions.SnapshotPath(dir, sanctions.SourceOFAC))
				require.NoError(t, err)
				before = b
			}

			r, reader := newGaugedRefresher(t, dir)
			core, obs := observer.New(zapcore.DebugLevel)
			r.log = zap.New(core)

			out := r.refresh(context.Background(), job{target: mustTarget(t, "ofac"), fetch: tt.fetch})
			require.Error(t, out.err, "the cause stays on the outcome")
			assert.Equal(t, tt.wantFallback, out.fallback)
			assert.Equal(t, tt.wantBlocked, out.blocked)
			if tt.wantBlocked {
				require.ErrorIs(t, out.err, sanctions.ErrStaleSnapshot)
				assert.Len(t, obs.FilterMessage("screening blocked: sanctions snapshot stale").All(), 1)
			}
			if tt.wantFallback {
				warn := obs.FilterMessage("upstream unreachable, serving cached snapshot").All()
				require.Len(t, warn, 1)
				assert.Equal(t, "ofac", warn[0].ContextMap()["list"])
				assert.Equal(t, tt.cachedAge, warn[0].ContextMap()["snapshot_age"])
			}

			// The cached snapshot is never touched by a failed refresh.
			if tt.cachedAge >= 0 {
				after, err := os.ReadFile(sanctions.SnapshotPath(dir, sanctions.SourceOFAC))
				require.NoError(t, err)
				assert.Equal(t, before, after)
				ages, _, ok := listAges(t, reader)
				require.True(t, ok)
				assert.InDelta(t, tt.cachedAge.Seconds(), ages["ofac"], 1e-9, "the age keeps being reported, so the alert sees it")
			}

			// run: a fallback is not a failure of the Job; blocked lists and bad formats are.
			r.age = nil
			err := r.run(context.Background(), []job{{target: mustTarget(t, "ofac"), fetch: tt.fetch}})
			if tt.wantRunErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// A run where one list falls back and another refreshes exits zero; the fallback list keeps its old snapshot.
func TestRun_FallbackListDoesNotFailTheRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, writeFixtureSnapshot(dir, "ofac", testNow.Add(-2*time.Hour)))
	r, _ := newTestRefresher(dir)
	jobs := []job{
		{target: mustTarget(t, "ofac"), fetch: unreachable()},
		{target: mustTarget(t, "un"), fetch: fileFetch(fixtures["un"])},
	}
	require.NoError(t, r.run(context.Background(), jobs))
	snap, err := sanctions.ReadSnapshot(dir, sanctions.SourceOFAC)
	require.NoError(t, err)
	assert.True(t, testNow.Add(-2*time.Hour).Equal(snap.FetchedAt))
	_, err = sanctions.ReadSnapshot(dir, sanctions.SourceUN)
	require.NoError(t, err)
}

// The stale error names the list and the age, which is what the on-call needs, and nothing else.
func TestRefresh_BlockedErrorNamesListAndAge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, writeFixtureSnapshot(dir, "eu", testNow.Add(-30*time.Hour)))
	r, _ := newTestRefresher(dir)
	out := r.refresh(context.Background(), job{target: mustTarget(t, "eu"), fetch: failingFetch(errors.New("boom"))})
	require.ErrorIs(t, out.err, sanctions.ErrStaleSnapshot)
	assert.Contains(t, out.err.Error(), "eu")
	assert.Contains(t, out.err.Error(), "30h0m0s")
}

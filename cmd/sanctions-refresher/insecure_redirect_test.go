package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// stripURL drops the URL and keeps the cause, so the sentinels below stay reachable with errors.Is.
func TestStripURL_KeepsSentinelDropsURL(t *testing.T) {
	t.Parallel()
	const target = "http://mirror.example.test/fsf.csv?token=s3cr3t-token"
	for _, sentinel := range []error{errInsecureRedirect, errTooManyRedirects} {
		ue := &url.Error{Op: "Get", URL: target, Err: sentinel}
		err := fmt.Errorf("%w: eu: download: %w", errUpstream, stripURL(ue))
		require.ErrorIs(t, err, sentinel)
		assert.NotContains(t, err.Error(), "mirror.example.test")
		assert.NotContains(t, err.Error(), "s3cr3t-token")
	}
}

// An https list URL redirected to http:// is an integrity signal, not an outage: with a fresh snapshot on disk the
// refresh still fails the run at Error level (no fallback), the snapshot keeps serving byte for byte, and neither the
// error nor the logs carry a URL or the token. A plain outage with the same snapshot still falls back.
func TestRefresh_InsecureRedirectIsNotAFallback(t *testing.T) {
	t.Parallel()
	const secret = "s3cr3t-token"
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "poisoned list")
	}))
	t.Cleanup(plain.Close)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/sdn.xml?token="+secret, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	gone := httptest.NewTLSServer(http.NotFoundHandler())
	goneURL := gone.URL
	gone.Close()

	tests := []struct {
		name         string
		url          string
		wantFallback bool
	}{
		{name: "redirect to http", url: srv.URL + "/sdn.xml?token=" + secret},
		{name: "unreachable (control)", url: goneURL + "/sdn.xml?token=" + secret, wantFallback: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, writeFixtureSnapshot(dir, "ofac", testNow.Add(-time.Hour)))
			before, err := os.ReadFile(sanctions.SnapshotPath(dir, sanctions.SourceOFAC))
			require.NoError(t, err)

			r, _ := newTestRefresher(dir)
			core, logs := observer.New(zapcore.DebugLevel)
			r.log = zap.New(core)
			j := job{target: mustTarget(t, "ofac"), fetch: httpFetch(trustingClient(t, srv), "ofac", tt.url, 1<<20)}

			out := r.refresh(context.Background(), j)
			require.Error(t, out.err)
			assert.Equal(t, tt.wantFallback, out.fallback)
			assert.False(t, out.blocked)
			runErr := r.run(context.Background(), []job{j})

			warn := logs.FilterMessage("upstream unreachable, serving cached snapshot").All()
			failed := logs.FilterMessage("list refresh failed").All()
			if tt.wantFallback {
				require.ErrorIs(t, out.err, errUpstream)
				assert.Len(t, warn, 2)
				assert.Empty(t, failed)
				require.NoError(t, runErr)
			} else {
				require.ErrorIs(t, out.err, errInsecureRedirect)
				assert.Empty(t, warn, "an insecure redirect must not be logged as an outage")
				require.Len(t, failed, 2)
				for _, e := range failed {
					assert.Equal(t, zapcore.ErrorLevel, e.Level)
				}
				require.Error(t, runErr)
				require.ErrorIs(t, runErr, errInsecureRedirect)
			}

			after, err := os.ReadFile(sanctions.SnapshotPath(dir, sanctions.SourceOFAC))
			require.NoError(t, err)
			assert.Equal(t, before, after, "the cached snapshot keeps serving untouched")

			texts := []string{out.err.Error()}
			if runErr != nil {
				texts = append(texts, runErr.Error())
			}
			for _, e := range logs.All() {
				texts = append(texts, e.Message, fmt.Sprint(e.ContextMap()))
			}
			for _, s := range texts {
				assert.NotContains(t, s, secret)
				assert.NotContains(t, s, strings.TrimPrefix(plain.URL, "http://"))
				assert.NotContains(t, s, strings.TrimPrefix(srv.URL, "https://"))
				assert.NotContains(t, s, strings.TrimPrefix(goneURL, "https://"))
			}
			assert.False(t, errors.Is(out.err, sanctions.ErrStaleSnapshot))
		})
	}
}

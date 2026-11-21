package main

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trustingClient is newHTTPClient (TLS 1.3, redirect policy) trusting the test server's certificate.
func trustingClient(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	c := newHTTPClient(time.Minute)
	tr, ok := c.Transport.(*http.Transport)
	require.True(t, ok)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	tr.TLSClientConfig.RootCAs = pool
	return c
}

// A redirect may only lead to another https URL, and only a few times: a 30x to http:// would download the list in
// clear, outside TLS 1.3, where it can be poisoned. The error names neither the target nor the original URL.
func TestHTTPFetch_RedirectPolicy(t *testing.T) {
	t.Parallel()
	const secret = "s3cr3t-token"
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "poisoned list")
	}))
	t.Cleanup(plain.Close)
	plainTarget := plain.URL + "/fsf.csv?token=" + secret

	var tlsURL string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/to-http":
			http.Redirect(w, r, plainTarget, http.StatusFound)
		case "/to-https":
			http.Redirect(w, r, tlsURL+"/ok?token="+secret, http.StatusMovedPermanently)
		case "/loop":
			http.Redirect(w, r, tlsURL+"/loop?token="+secret, http.StatusFound)
		case "/ok":
			_, _ = io.WriteString(w, "list")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	tlsURL = srv.URL

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr error
	}{
		{name: "https to https is followed", path: "/to-https", want: "list"},
		{name: "https to http is refused", path: "/to-http", wantErr: errInsecureRedirect},
		{name: "endless redirects stop", path: "/loop", wantErr: errTooManyRedirects},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fetch := httpFetch(trustingClient(t, srv), "eu", srv.URL+tt.path+"?token="+secret, 1<<20)
			rc, err := fetch(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				msg := err.Error()
				assert.NotContains(t, msg, secret)
				assert.NotContains(t, msg, plain.URL, "the redirect target must not reach the error")
				assert.NotContains(t, msg, strings.TrimPrefix(plain.URL, "http://"))
				assert.NotContains(t, msg, strings.TrimPrefix(srv.URL, "https://"), "nor the original URL")
				return
			}
			require.NoError(t, err)
			defer func() { _ = rc.Close() }()
			b, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}

func TestCheckRedirect(t *testing.T) {
	t.Parallel()
	mk := func(raw string) *http.Request {
		r, err := http.NewRequest(http.MethodGet, raw, nil)
		require.NoError(t, err)
		return r
	}
	via := func(n int) []*http.Request {
		out := make([]*http.Request, n)
		for i := range out {
			out[i] = mk("https://list.example.test/")
		}
		return out
	}
	tests := []struct {
		name string
		req  *http.Request
		via  []*http.Request
		want error
	}{
		{"https, first hop", mk("https://cdn.example.test/list.xml"), via(1), nil},
		{"https, last allowed hop", mk("https://cdn.example.test/list.xml"), via(maxRedirects - 1), nil},
		{"https, one hop too many", mk("https://cdn.example.test/list.xml"), via(maxRedirects), errTooManyRedirects},
		{"http", mk("http://cdn.example.test/list.xml"), via(1), errInsecureRedirect},
		{"ftp", mk("ftp://cdn.example.test/list.xml"), via(1), errInsecureRedirect},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkRedirect(tt.req, tt.via)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
			assert.NotContains(t, err.Error(), "cdn.example.test")
		})
	}
}

func TestNewHTTPClient_RedirectPolicyWired(t *testing.T) {
	t.Parallel()
	c := newHTTPClient(time.Minute)
	require.NotNil(t, c.CheckRedirect)
	req, err := http.NewRequest(http.MethodGet, "http://cdn.example.test/", nil)
	require.NoError(t, err)
	require.ErrorIs(t, c.CheckRedirect(req, nil), errInsecureRedirect)
}

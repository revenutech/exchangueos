//go:build e2e

// Package e2e — shared harness for end-to-end tests against the local docker-compose stack.
//
// Tests are guarded by `//go:build e2e`. Run via `task test:e2e` after
// `task compose:up` has brought the stack to a healthy state.
//
// Helpers focus on assertion ergonomics:
//
//   - Eventually(t, condition, timeout, interval) — polling assertion with deadline
//   - NewClient() — http.Client targeting the local exchangeos-api on :8084
//   - SeedTenant(t) — POST or DB insert of a fresh tenant for the test
//
// NO time.Sleep — always use Eventually with explicit timeout.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8084"
	defaultTimeout = 30 * time.Second
)

// BaseURL returns the API base URL — overridable via EXCHANGEOS_E2E_BASE_URL.
func BaseURL() string {
	if v := os.Getenv("EXCHANGEOS_E2E_BASE_URL"); v != "" {
		return v
	}
	return defaultBaseURL
}

// NewClient returns an http.Client with a 10s timeout, suitable for E2E calls.
func NewClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// WaitHealthy polls /healthz until 200 or timeout. Use at suite start.
func WaitHealthy(t *testing.T, timeout time.Duration) {
	t.Helper()
	c := NewClient()
	Eventually(t, func() bool {
		resp, err := c.Get(BaseURL() + "/healthz")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, timeout, 250*time.Millisecond, "exchangeos-api not healthy")
}

// Eventually polls `cond` until it returns true or timeout. Fails the test on timeout.
func Eventually(t *testing.T, cond func() bool, timeout, interval time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("Eventually deadline exceeded: %s", msg)
}

// GET issues a GET against the API and unmarshals the JSON body into `out`.
func GET(t *testing.T, path string, out interface{}) (int, error) {
	t.Helper()
	c := NewClient()
	resp, err := c.Get(BaseURL() + path)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(body, out); err != nil {
			return resp.StatusCode, fmt.Errorf("unmarshal: %w; body=%s", err, body)
		}
	}
	return resp.StatusCode, nil
}

// GETQuery is GET with query params.
func GETQuery(t *testing.T, path string, params url.Values, out interface{}) (int, error) {
	t.Helper()
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return GET(t, path, out)
}

// Ctx returns a 30s context — most E2E ops should complete well within this.
func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

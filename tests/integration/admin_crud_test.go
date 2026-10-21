//go:build integration

// Package integration — admin CRUD round-trip against the live local stack.
//
// Run via:
//   go test -tags integration ./tests/integration/admin_crud_test.go
//
// Requires the local docker compose stack up (CRDB + api with
// EXCHANGEOS_ENABLE_ADMIN_API=true). The smoke variant
// `scripts/smoke-crud.sh` covers LIST-only; this test exercises the
// POST → GET → PUT → GET → DELETE cycle end-to-end via the live HTTP API.
package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultBase = "http://localhost:8084"

func baseURL() string {
	if v := os.Getenv("EXCHANGEOS_BASE_URL"); v != "" {
		return v
	}
	return defaultBase
}

func httpJSON(t *testing.T, method, url string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

func TestAdminCRUD_Currency_FullLifecycle(t *testing.T) {
	base := baseURL()
	code := "XCD" // ISO 4217 placeholder safe for tests (Special Reference reserved range)
	url := fmt.Sprintf("%s/v1/admin/currencies", base)
	idURL := url + "/" + code

	t.Cleanup(func() { _, _ = httpJSON(t, "DELETE", idURL, nil) })

	// 1) POST
	status, body := httpJSON(t, "POST", url, map[string]any{
		"code":           code,
		"name":           "Test Carib Dollar (admin-crud test)",
		"minor_units":    2,
		"cls_eligible":   false,
		"cfets_eligible": false,
		"active":         true,
	})
	require.Equal(t, http.StatusCreated, status, "POST: %v", body)
	assert.Equal(t, "created", body["status"])

	// 2) GET — confirm it was inserted
	status, body = httpJSON(t, "GET", idURL, nil)
	require.Equal(t, http.StatusOK, status, "GET: %v", body)
	assert.Equal(t, code, body["code"])
	assert.Equal(t, true, body["active"])

	// 3) PUT — flip active=false
	status, body = httpJSON(t, "PUT", idURL, map[string]any{"active": false})
	require.Equal(t, http.StatusOK, status, "PUT: %v", body)

	// 4) GET — confirm mutation
	status, body = httpJSON(t, "GET", idURL, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, false, body["active"])

	// 5) DELETE
	status, body = httpJSON(t, "DELETE", idURL, nil)
	require.Equal(t, http.StatusOK, status, "DELETE: %v", body)

	// 6) GET after delete → 404
	status, _ = httpJSON(t, "GET", idURL, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminCRUD_AuditEvents_AreReadOnly(t *testing.T) {
	base := baseURL()
	url := fmt.Sprintf("%s/v1/admin/audit-events", base)

	// LIST works
	status, body := httpJSON(t, "GET", url+"?limit=2", nil)
	require.Equal(t, http.StatusOK, status)
	assert.GreaterOrEqual(t, int(body["count"].(float64)), 1)

	// POST refused
	status, _ = httpJSON(t, "POST", url, map[string]any{"event_type": "TEST"})
	assert.Equal(t, http.StatusMethodNotAllowed, status, "audit_events must be read-only")
}

func TestAdminCRUD_OutboxArchive_AreReadOnly(t *testing.T) {
	base := baseURL()
	url := fmt.Sprintf("%s/v1/admin/outbox-archive", base)

	status, body := httpJSON(t, "GET", url+"?limit=2", nil)
	require.Equal(t, http.StatusOK, status)
	assert.NotNil(t, body["count"])

	// DELETE refused
	status, _ = httpJSON(t, "DELETE", url+"/17171717-0000-5000-8000-000000000001", nil)
	assert.Equal(t, http.StatusMethodNotAllowed, status, "archive must be read-only")
}

func TestAdminCRUD_UnknownTable_Returns404(t *testing.T) {
	base := baseURL()
	status, _ := httpJSON(t, "GET", base+"/v1/admin/no-such-table", nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestAdminCRUD_Filter_Works(t *testing.T) {
	base := baseURL()

	// fx-trades?status=SETTLED should return exactly 1 (seed 08 has one SETTLED swap near leg).
	status, body := httpJSON(t, "GET", base+"/v1/admin/fx-trades?status=SETTLED", nil)
	require.Equal(t, http.StatusOK, status)
	count := int(body["count"].(float64))
	assert.Equal(t, 1, count, "expected exactly 1 SETTLED trade from seed 08")

	// fx-trades?status=CONFIRMED should return 4.
	status, body = httpJSON(t, "GET", base+"/v1/admin/fx-trades?status=CONFIRMED", nil)
	require.Equal(t, http.StatusOK, status)
	count = int(body["count"].(float64))
	assert.Equal(t, 4, count, "expected exactly 4 CONFIRMED trades")
}

func TestAdminCRUD_Pagination(t *testing.T) {
	base := baseURL()

	// LIST with limit=3 + offset=0 then limit=3 + offset=3 should return non-overlapping sets.
	status, body1 := httpJSON(t, "GET", base+"/v1/admin/currencies?limit=3&offset=0", nil)
	require.Equal(t, http.StatusOK, status)

	status, body2 := httpJSON(t, "GET", base+"/v1/admin/currencies?limit=3&offset=3", nil)
	require.Equal(t, http.StatusOK, status)

	items1 := body1["items"].([]any)
	items2 := body2["items"].([]any)
	require.Len(t, items1, 3)
	require.Len(t, items2, 3)

	// First item of page 2 must differ from first item of page 1.
	first1 := items1[0].(map[string]any)["code"]
	first2 := items2[0].(map[string]any)["code"]
	assert.NotEqual(t, first1, first2, "pagination overlap detected")
}

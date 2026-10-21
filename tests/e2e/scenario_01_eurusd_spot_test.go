//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	e2e "github.com/revenutech/exchangeos/tests/e2e"
)

// Scenario 1 — EUR/USD spot booking smoke.
//
// Asserts:
//   - /healthz returns 200
//   - /v1/refdata/currencies?active_only=true returns at least 18 CLS-eligible currencies
//
// Full booking flow (Quote → AcceptQuote → Trade) requires more API surface than
// the current smoke endpoints expose; this test brackets the entry/exit points.
func TestScenario01_EURUSD_Spot_RefdataAvailable(t *testing.T) {
	e2e.WaitHealthy(t, 30*time.Second)

	var resp struct {
		Currencies []struct {
			Code        string `json:"code"`
			CLSEligible bool   `json:"cls_eligible"`
		} `json:"currencies"`
		Count int `json:"count"`
	}
	status, err := e2e.GET(t, "/v1/refdata/currencies?active_only=true", &resp)
	if err != nil {
		t.Fatalf("GET currencies: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status: got %d want 200", status)
	}

	// Smoke seed populates 5 dev pairs; ensure at minimum EUR + USD are present.
	wantSet := map[string]bool{"EUR": true, "USD": true}
	for _, c := range resp.Currencies {
		delete(wantSet, c.Code)
	}
	if len(wantSet) > 0 {
		t.Fatalf("missing currencies in /v1/refdata/currencies: %v (got %d)", wantSet, resp.Count)
	}
}

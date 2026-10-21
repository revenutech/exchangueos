//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	e2e "github.com/revenutech/exchangeos/tests/e2e"
)

// Scenario 5 — Risk limit breach pre-trade.
//
// Validates the smoke API surface. Full CheckLimit + Reserve flow requires the
// gRPC adapter under -tags grpcgen; this test asserts the prerequisites:
//
//   - API readiness
//   - /v1/trades/:id returns 404 for non-existent ids (correct error code)
//
// Risk gRPC happy path is exercised by unit tests in modules/risk/application;
// the integration-side validation lives here once the public REST surface lands.
func TestScenario05_NonExistentTrade_Returns404(t *testing.T) {
	e2e.WaitHealthy(t, 30*time.Second)

	status, err := e2e.GET(t, "/v1/trades/00000000-0000-0000-0000-000000000001", nil)
	if err != nil {
		t.Fatalf("GET non-existent trade: %v", err)
	}
	if status != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", status)
	}
}

// Scenario 5b — invalid trade-id returns 400.
func TestScenario05b_MalformedTradeID_Returns400(t *testing.T) {
	e2e.WaitHealthy(t, 30*time.Second)
	status, _ := e2e.GET(t, "/v1/trades/not-a-uuid", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", status)
	}
}

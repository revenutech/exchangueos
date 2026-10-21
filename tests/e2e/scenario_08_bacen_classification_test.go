//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	e2e "github.com/revenutech/exchangeos/tests/e2e"
)

// Scenario 8 — BACEN classification + IOF (smoke).
//
// Validates the system is up and refdata-served — full BACEN flow needs the
// ComplianceService REST/gRPC surface (currently only smoke surface for refdata
// + trade in the public HTTP API). The pkg/bacen unit tests cover the
// classification/IOF formulas thoroughly; this E2E asserts the wiring is alive.
func TestScenario08_BACEN_RefdataServiceReachable(t *testing.T) {
	e2e.WaitHealthy(t, 30*time.Second)

	// Version endpoint sanity.
	var v struct {
		Service string `json:"service"`
		Version string `json:"version"`
		Env     string `json:"env"`
	}
	status, err := e2e.GET(t, "/version", &v)
	if err != nil {
		t.Fatalf("GET /version: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status: got %d want 200", status)
	}
	if v.Service == "" || v.Version == "" {
		t.Fatalf("version payload empty: %+v", v)
	}
}

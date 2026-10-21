package pricing_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// almostEqual compares with tolerance to absorb the 8-decimal banker rounding.
func almostEqual(t *testing.T, got, want decimal.Decimal, tol string) {
	t.Helper()
	tolerance := decimal.RequireFromString(tol)
	if got.Sub(want).Abs().GreaterThan(tolerance) {
		t.Fatalf("got %s, want ≈ %s (±%s)", got, want, tol)
	}
}

// ─── Golden cases (cite source) ────────────────────────────────────────────

// EUR/USD 90 days, US 5.25%, EU 4.00%, both 360 basis.
//
//	F = 1.0800 × (1 + 0.0525 × 90/360) / (1 + 0.0400 × 90/360)
//	  = 1.0800 × 1.013125 / 1.010000
//	  = 1.094175 / 1.010000
//	  = 1.08334158415841584158...  → RoundBank(8) = 1.08334158
func TestForward_EURUSD_90d_Golden(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.0800"),
		QuotedRate:  dec("0.0525"),
		BaseRate:    dec("0.0400"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, err := pricing.Forward(in)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	almostEqual(t, f, dec("1.08334158"), "0.00000001")
}

// USDBRL 30 days, BR rate 11.75% (basis 252→approximated as 365 for this synthetic test),
// US rate 5.25%, both 360 basis. Hand calc:
// F = 5.0000 × (1 + 0.1175 × 30/360) / (1 + 0.0525 × 30/360)
//
//	= 5.0000 × 1.00979166666... / 1.00437500000
//	= 5.04895833333... / 1.004375
//	= 5.02696535988384152665...  → RoundBank(8) = 5.02696536
func TestForward_USDBRL_30d_Synthetic(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("5.0000"),
		QuotedRate:  dec("0.1175"),
		BaseRate:    dec("0.0525"),
		Days:        30,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, err := pricing.Forward(in)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	almostEqual(t, f, dec("5.02696536"), "0.00000005")
}

// GBPUSD 180d — GBP uses 365 basis (sterling money-market convention).
// F = 1.2700 × (1 + 0.0525 × 180/360) / (1 + 0.0475 × 180/365)
//
//	= 1.2700 × 1.02625 / 1.02342465753424657534...
//	= 1.3033375 / 1.02342465753424657534...
//	= 1.27350605675277740596...  → RoundBank(8) = 1.27350606
func TestForward_GBPUSD_180d_MixedBasis(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.2700"),
		QuotedRate:  dec("0.0525"),
		BaseRate:    dec("0.0475"),
		Days:        180,
		QuotedBasis: 360,
		BaseBasis:   365,
	}
	f, err := pricing.Forward(in)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	almostEqual(t, f, dec("1.27350606"), "0.00000005")
}

// USDJPY 365d, US 5.25%, JP 0.10%
// F = 145.00 × (1 + 0.0525 × 365/360) / (1 + 0.0010 × 365/360)
//
//	= 145.00 × 1.05322916666... / 1.00101388888...
//	= 152.71822916666... / 1.00101388888...
//	= 152.56354668183647135543...  → RoundBank(8) = 152.56354668
func TestForward_USDJPY_365d_LargeSpread(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("145.00"),
		QuotedRate:  dec("0.0525"),
		BaseRate:    dec("0.0010"),
		Days:        365,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, err := pricing.Forward(in)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	almostEqual(t, f, dec("152.56354668"), "0.00000010")
}

// ─── Properties ────────────────────────────────────────────────────────────

// At days=0, forward equals spot (no interest accrual).
func TestForward_ZeroDays_EqualsSpot(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.2345"),
		QuotedRate:  dec("0.05"),
		BaseRate:    dec("0.02"),
		Days:        0,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, err := pricing.Forward(in)
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	if !f.Equal(dec("1.23450000")) {
		t.Fatalf("got %s want 1.23450000", f)
	}
}

// If i_p == i_b and bases match, forward == spot.
func TestForward_EqualRatesEqualBasis_NoPoints(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.10"),
		QuotedRate:  dec("0.03"),
		BaseRate:    dec("0.03"),
		Days:        180,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, _ := pricing.Forward(in)
	if !f.Equal(dec("1.10000000")) {
		t.Fatalf("got %s want 1.10000000", f)
	}
	pts, _ := pricing.ForwardPoints(in)
	if !pts.IsZero() {
		t.Fatalf("expected zero points, got %s", pts)
	}
}

// If quoted rate > base rate → forward > spot (positive carry).
func TestForward_PositiveCarry(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.0000"),
		QuotedRate:  dec("0.06"),
		BaseRate:    dec("0.02"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, _ := pricing.Forward(in)
	if !f.GreaterThan(dec("1.0000")) {
		t.Fatalf("expected forward > spot, got %s", f)
	}
}

// If quoted rate < base rate → forward < spot (negative carry).
func TestForward_NegativeCarry(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.0000"),
		QuotedRate:  dec("0.01"),
		BaseRate:    dec("0.05"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, _ := pricing.Forward(in)
	if !f.LessThan(dec("1.0000")) {
		t.Fatalf("expected forward < spot, got %s", f)
	}
}

// ─── ForwardPoints sanity ──────────────────────────────────────────────────

func TestForwardPoints_EqualsForwardMinusSpot(t *testing.T) {
	in := pricing.ForwardInput{
		Spot:        dec("1.0800"),
		QuotedRate:  dec("0.0525"),
		BaseRate:    dec("0.0400"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	f, _ := pricing.Forward(in)
	pts, _ := pricing.ForwardPoints(in)
	if !pts.Equal(f.Sub(in.Spot)) {
		t.Fatalf("points %s != F − S %s", pts, f.Sub(in.Spot))
	}
}

// ─── Validation ────────────────────────────────────────────────────────────

func TestForward_RejectsBadInputs(t *testing.T) {
	base := pricing.ForwardInput{
		Spot:        dec("1.0800"),
		QuotedRate:  dec("0.05"),
		BaseRate:    dec("0.02"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	cases := []struct {
		name   string
		mutate func(*pricing.ForwardInput)
	}{
		{"zero spot", func(in *pricing.ForwardInput) { in.Spot = decimal.Zero }},
		{"negative spot", func(in *pricing.ForwardInput) { in.Spot = dec("-1") }},
		{"negative quoted rate", func(in *pricing.ForwardInput) { in.QuotedRate = dec("-0.01") }},
		{"negative base rate", func(in *pricing.ForwardInput) { in.BaseRate = dec("-0.01") }},
		{"negative days", func(in *pricing.ForwardInput) { in.Days = -1 }},
		{"bad quoted basis", func(in *pricing.ForwardInput) { in.QuotedBasis = 100 }},
		{"bad base basis", func(in *pricing.ForwardInput) { in.BaseBasis = 100 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			_, err := pricing.Forward(in)
			if !errors.Is(err, pricing.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── Benchmark ─────────────────────────────────────────────────────────────

func BenchmarkForwardSimple(b *testing.B) {
	in := pricing.ForwardInput{
		Spot:        dec("1.0800"),
		QuotedRate:  dec("0.0525"),
		BaseRate:    dec("0.0400"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pricing.Forward(in)
	}
}

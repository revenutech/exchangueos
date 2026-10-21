package pricing_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

// BRL devalues 5.00 → 5.10 with 1M BRL notional.
// Dealer (long BRL) loses: 1M × (1/5.10 − 1/5.00) ≈ −3,921.57 USD.
func TestNDF_BRLDevaluation(t *testing.T) {
	in := pricing.NDFInput{
		NotionalReferenceCCY: decimal.NewFromInt(1_000_000),
		ContractRate:         dec("5.00"),
		FixingRate:           dec("5.10"),
	}
	got, err := pricing.SettlementAmount(in)
	if err != nil {
		t.Fatalf("SettlementAmount: %v", err)
	}
	want := dec("-3921.56862745")
	if got.Sub(want).Abs().GreaterThan(dec("0.00000005")) {
		t.Fatalf("got %s want ≈ %s", got, want)
	}
	if !got.IsNegative() {
		t.Fatalf("expected negative settlement (dealer pays), got %s", got)
	}
}

// BRL appreciates 5.00 → 4.90 with 1M BRL notional.
// Dealer (long BRL) gains: 1M × (1/4.90 − 1/5.00) ≈ +4,081.63 USD.
func TestNDF_BRLAppreciation(t *testing.T) {
	in := pricing.NDFInput{
		NotionalReferenceCCY: decimal.NewFromInt(1_000_000),
		ContractRate:         dec("5.00"),
		FixingRate:           dec("4.90"),
	}
	got, err := pricing.SettlementAmount(in)
	if err != nil {
		t.Fatalf("SettlementAmount: %v", err)
	}
	want := dec("4081.63265306")
	if got.Sub(want).Abs().GreaterThan(dec("0.00000005")) {
		t.Fatalf("got %s want ≈ %s", got, want)
	}
	if !got.IsPositive() {
		t.Fatalf("expected positive settlement (dealer receives), got %s", got)
	}
}

// At-the-money: fixing == contract → settlement = 0.
func TestNDF_ATMSettlesZero(t *testing.T) {
	in := pricing.NDFInput{
		NotionalReferenceCCY: decimal.NewFromInt(1_000_000),
		ContractRate:         dec("5.00"),
		FixingRate:           dec("5.00"),
	}
	got, _ := pricing.SettlementAmount(in)
	if !got.IsZero() {
		t.Fatalf("ATM should settle zero, got %s", got)
	}
}

// Symmetry: opposite moves of identical magnitude do NOT yield exactly equal absolute settlements
// (the formula is non-linear), but the SIGN should always flip and the magnitudes should be in
// the right ballpark. This test asserts the sign property and bounded magnitude difference.
func TestNDF_SymmetryProperty(t *testing.T) {
	up := pricing.NDFInput{
		NotionalReferenceCCY: decimal.NewFromInt(1_000_000),
		ContractRate:         dec("5.00"),
		FixingRate:           dec("5.10"),
	}
	down := pricing.NDFInput{
		NotionalReferenceCCY: decimal.NewFromInt(1_000_000),
		ContractRate:         dec("5.00"),
		FixingRate:           dec("4.90"),
	}
	a, _ := pricing.SettlementAmount(up)
	b, _ := pricing.SettlementAmount(down)
	if a.IsPositive() || b.IsNegative() {
		t.Fatalf("expected sign flip: up=%s down=%s", a, b)
	}
	// Magnitudes should differ by less than 10% for a 2% rate move.
	ratio := a.Abs().Div(b.Abs())
	low := dec("0.90")
	high := dec("1.10")
	if ratio.LessThan(low) || ratio.GreaterThan(high) {
		t.Fatalf("magnitude ratio out of range [0.90, 1.10]: %s (a=%s b=%s)", ratio, a, b)
	}
}

func TestNDF_RejectsBadInputs(t *testing.T) {
	base := pricing.NDFInput{
		NotionalReferenceCCY: decimal.NewFromInt(1_000_000),
		ContractRate:         dec("5.00"),
		FixingRate:           dec("5.10"),
	}
	cases := []struct {
		name   string
		mutate func(*pricing.NDFInput)
	}{
		{"zero notional", func(in *pricing.NDFInput) { in.NotionalReferenceCCY = decimal.Zero }},
		{"negative notional", func(in *pricing.NDFInput) { in.NotionalReferenceCCY = dec("-1") }},
		{"zero contract", func(in *pricing.NDFInput) { in.ContractRate = decimal.Zero }},
		{"negative contract", func(in *pricing.NDFInput) { in.ContractRate = dec("-5") }},
		{"zero fixing", func(in *pricing.NDFInput) { in.FixingRate = decimal.Zero }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			_, err := pricing.SettlementAmount(in)
			if !errors.Is(err, pricing.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

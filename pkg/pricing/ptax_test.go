package pricing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

func validPTAX(t *testing.T) pricing.PTAX {
	t.Helper()
	return pricing.PTAX{
		Date: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		Windows: [4]pricing.PTAXWindow{
			{Hour: 10, Bid: dec("5.1010"), Ask: dec("5.1030")},
			{Hour: 11, Bid: dec("5.1020"), Ask: dec("5.1040")},
			{Hour: 12, Bid: dec("5.1015"), Ask: dec("5.1035")},
			{Hour: 13, Bid: dec("5.1025"), Ask: dec("5.1045")},
		},
	}
}

// Mid per window: (5.1020, 5.1030, 5.1025, 5.1035) → mean = 5.10275
// Banker's round to 4 decimals → 5.1028 (.5 → even, but 5 → 8 because last kept digit is 2 (even); .75 rounds up to even 8).
func TestPTAX_WeightedFixing(t *testing.T) {
	got, err := validPTAX(t).WeightedFixing()
	if err != nil {
		t.Fatalf("WeightedFixing: %v", err)
	}
	want := dec("5.1028") // banker rounding to 4 decimals
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

// Bid mean: (5.1010+5.1020+5.1015+5.1025)/4 = 5.10175 → banker rounds .5 to even (7→8) → 5.1018
// Ask mean: (5.1030+5.1040+5.1035+5.1045)/4 = 5.10375 → 5.1038
func TestPTAX_BidAskFixing(t *testing.T) {
	p := validPTAX(t)
	bid, err := p.BidFixing()
	if err != nil {
		t.Fatalf("BidFixing: %v", err)
	}
	if !bid.Equal(dec("5.1018")) {
		t.Errorf("bid: got %s want 5.1018", bid)
	}
	ask, err := p.AskFixing()
	if err != nil {
		t.Fatalf("AskFixing: %v", err)
	}
	if !ask.Equal(dec("5.1038")) {
		t.Errorf("ask: got %s want 5.1038", ask)
	}
}

func TestPTAX_RejectsBadInput(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*pricing.PTAX)
	}{
		{"zero date", func(p *pricing.PTAX) { p.Date = time.Time{} }},
		{"bad hour", func(p *pricing.PTAX) { p.Windows[2].Hour = 99 }},
		{"zero bid", func(p *pricing.PTAX) { p.Windows[0].Bid = dec("0") }},
		{"negative ask", func(p *pricing.PTAX) { p.Windows[1].Ask = dec("-1") }},
		{"bid > ask", func(p *pricing.PTAX) { p.Windows[3].Bid = dec("99"); p.Windows[3].Ask = dec("1") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPTAX(t)
			tc.mutate(&p)
			if _, err := p.WeightedFixing(); !errors.Is(err, pricing.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestPTAX_FetcherFunc_AdaptsClosure(t *testing.T) {
	called := false
	f := pricing.PTAXFetcherFunc(func(ctx context.Context, _ time.Time) (pricing.PTAX, error) {
		called = true
		return validPTAX(t), nil
	})
	p, err := f.FetchPTAX(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("FetchPTAX: %v", err)
	}
	if !called {
		t.Fatal("closure not invoked")
	}
	if p.Date.IsZero() {
		t.Fatal("expected populated PTAX")
	}
}

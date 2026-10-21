package pricing_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

func TestPositionMTM_LongPositive(t *testing.T) {
	p := pricing.Position{
		NotionalBase: decimal.NewFromInt(1_000_000),
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		DealRate:     dec("1.0800"),
		MarketRate:   dec("1.0850"),
		Side:         pricing.SideLong,
	}
	pnl, err := pricing.PositionMTM(p)
	if err != nil {
		t.Fatalf("PositionMTM: %v", err)
	}
	if !pnl.Equal(dec("5000")) {
		t.Fatalf("got %s want 5000", pnl)
	}
}

func TestPositionMTM_LongNegative(t *testing.T) {
	p := pricing.Position{
		NotionalBase: decimal.NewFromInt(1_000_000),
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		DealRate:     dec("1.0800"),
		MarketRate:   dec("1.0750"),
		Side:         pricing.SideLong,
	}
	pnl, _ := pricing.PositionMTM(p)
	if !pnl.Equal(dec("-5000")) {
		t.Fatalf("got %s want -5000", pnl)
	}
}

func TestPositionMTM_ShortIsOppositeOfLong(t *testing.T) {
	long := pricing.Position{
		NotionalBase: decimal.NewFromInt(500_000),
		BaseCCY:      "GBP",
		QuoteCCY:     "USD",
		DealRate:     dec("1.27"),
		MarketRate:   dec("1.29"),
		Side:         pricing.SideLong,
	}
	short := long
	short.Side = pricing.SideShort

	pl, _ := pricing.PositionMTM(long)
	ps, _ := pricing.PositionMTM(short)
	if !pl.Add(ps).IsZero() {
		t.Fatalf("LONG + SHORT should net to zero: long=%s short=%s sum=%s", pl, ps, pl.Add(ps))
	}
}

func TestPositionMTM_ZeroMoveZeroPNL(t *testing.T) {
	p := pricing.Position{
		NotionalBase: decimal.NewFromInt(1_000_000),
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		DealRate:     dec("1.0800"),
		MarketRate:   dec("1.0800"),
		Side:         pricing.SideLong,
	}
	pnl, _ := pricing.PositionMTM(p)
	if !pnl.IsZero() {
		t.Fatalf("zero move should yield zero P&L, got %s", pnl)
	}
}

func TestPositionMTM_RejectsBadInputs(t *testing.T) {
	base := pricing.Position{
		NotionalBase: decimal.NewFromInt(1000),
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		DealRate:     dec("1.08"),
		MarketRate:   dec("1.09"),
		Side:         pricing.SideLong,
	}
	cases := []struct {
		name   string
		mutate func(*pricing.Position)
	}{
		{"zero notional", func(p *pricing.Position) { p.NotionalBase = decimal.Zero }},
		{"negative notional", func(p *pricing.Position) { p.NotionalBase = dec("-1") }},
		{"missing base", func(p *pricing.Position) { p.BaseCCY = "" }},
		{"same ccy", func(p *pricing.Position) { p.QuoteCCY = p.BaseCCY }},
		{"zero deal", func(p *pricing.Position) { p.DealRate = decimal.Zero }},
		{"zero market", func(p *pricing.Position) { p.MarketRate = decimal.Zero }},
		{"bad side", func(p *pricing.Position) { p.Side = "FLAT" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			_, err := pricing.PositionMTM(p)
			if !errors.Is(err, pricing.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestPortfolioMTM_AggregatesByQuoteCCY(t *testing.T) {
	positions := []pricing.Position{
		{
			NotionalBase: decimal.NewFromInt(1_000_000),
			BaseCCY:      "EUR", QuoteCCY: "USD",
			DealRate: dec("1.0800"), MarketRate: dec("1.0850"),
			Side: pricing.SideLong,
		},
		{
			NotionalBase: decimal.NewFromInt(500_000),
			BaseCCY:      "GBP", QuoteCCY: "USD",
			DealRate: dec("1.2700"), MarketRate: dec("1.2750"),
			Side: pricing.SideLong,
		},
		{
			NotionalBase: decimal.NewFromInt(2_000_000),
			BaseCCY:      "USD", QuoteCCY: "BRL",
			DealRate: dec("5.0000"), MarketRate: dec("5.1000"),
			Side: pricing.SideLong,
		},
	}
	pnl, err := pricing.PortfolioMTM(positions)
	if err != nil {
		t.Fatalf("PortfolioMTM: %v", err)
	}
	// USD bucket: 1M × 0.005 = 5000 + 500k × 0.005 = 2500 → 7500
	if !pnl["USD"].Equal(dec("7500")) {
		t.Errorf("USD bucket: got %s want 7500", pnl["USD"])
	}
	// BRL bucket: 2M × 0.10 = 200000
	if !pnl["BRL"].Equal(dec("200000")) {
		t.Errorf("BRL bucket: got %s want 200000", pnl["BRL"])
	}
	if len(pnl) != 2 {
		t.Errorf("buckets: got %d want 2", len(pnl))
	}
}

func TestPortfolioMTM_BadPositionPropagatesError(t *testing.T) {
	bad := []pricing.Position{
		{NotionalBase: decimal.NewFromInt(1000), BaseCCY: "EUR", QuoteCCY: "USD",
			DealRate: dec("1.08"), MarketRate: dec("1.09"), Side: pricing.SideLong},
		{NotionalBase: decimal.Zero, BaseCCY: "EUR", QuoteCCY: "USD",
			DealRate: dec("1.08"), MarketRate: dec("1.09"), Side: pricing.SideLong},
	}
	_, err := pricing.PortfolioMTM(bad)
	if !errors.Is(err, pricing.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

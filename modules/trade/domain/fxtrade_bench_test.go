package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/trade/domain"
)

// BenchmarkNewFXTrade measures aggregate construction cost (validation + event recording).
// Target: < 10µs on commodity hardware. Hot path on every BookTrade request.
func BenchmarkNewFXTrade(b *testing.B) {
	tradeDate := time.Date(2026, 5, 24, 14, 0, 0, 0, time.UTC)
	tenantID := uuid.New()
	in := domain.NewTradeInput{
		TenantID:       tenantID,
		ExternalRef:    "REF-001",
		TradeType:      domain.TradeTypeSpot,
		Venue:          domain.VenueCLS,
		BuyerBIC:       "DEUTDEFF",
		SellerBIC:      "CHASUS33",
		BoughtCurrency: "EUR",
		BoughtAmount:   decimal.NewFromInt(1_000_000),
		SoldCurrency:   "USD",
		SoldAmount:     decimal.NewFromInt(1_080_000),
		DealRate:       decimal.RequireFromString("1.08"),
		TradeDate:      tradeDate,
		ValueDate:      tradeDate.Add(48 * time.Hour),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := domain.NewFXTrade(in)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLifecycle_BookConfirmSettle measures the full happy-path lifecycle
// (NewFXTrade → Confirm → MarkSettling → MarkSettled). Reflects the request →
// settle latency budget at the domain layer.
func BenchmarkLifecycle_BookConfirmSettle(b *testing.B) {
	tradeDate := time.Date(2026, 5, 24, 14, 0, 0, 0, time.UTC)
	in := domain.NewTradeInput{
		TenantID:       uuid.New(),
		ExternalRef:    "REF-001",
		TradeType:      domain.TradeTypeSpot,
		Venue:          domain.VenueCLS,
		BuyerBIC:       "DEUTDEFF",
		SellerBIC:      "CHASUS33",
		BoughtCurrency: "EUR",
		BoughtAmount:   decimal.NewFromInt(1_000_000),
		SoldCurrency:   "USD",
		SoldAmount:     decimal.NewFromInt(1_080_000),
		DealRate:       decimal.RequireFromString("1.08"),
		TradeDate:      tradeDate,
		ValueDate:      tradeDate.Add(48 * time.Hour),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t, _ := domain.NewFXTrade(in)
		_ = t.Confirm()
		_ = t.MarkSettling()
		_ = t.MarkSettled("CLS-REF-" + t.ID().String())
	}
}

package pricing_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

func almostEqualP(t *testing.T, got, want decimal.Decimal, tol string) {
	t.Helper()
	tolerance := decimal.RequireFromString(tol)
	if got.Sub(want).Abs().GreaterThan(tolerance) {
		t.Fatalf("got %s, want ≈ %s (±%s)", got, want, tol)
	}
}

// EURUSD = 1.0800, USDBRL = 5.2000 → EURBRL = 5.6160
func TestCross_EUR_USD_BRL(t *testing.T) {
	a := pricing.Pair{BaseCCY: "EUR", QuoteCCY: "USD", Rate: dec("1.0800")}
	b := pricing.Pair{BaseCCY: "USD", QuoteCCY: "BRL", Rate: dec("5.2000")}
	got, err := pricing.Cross(a, b)
	if err != nil {
		t.Fatalf("Cross: %v", err)
	}
	if got.BaseCCY != "EUR" || got.QuoteCCY != "BRL" {
		t.Fatalf("pair: %s/%s want EUR/BRL", got.BaseCCY, got.QuoteCCY)
	}
	almostEqualP(t, got.Rate, dec("5.6160"), "0.00000005")
}

// EURUSD = 1.0800, BRLUSD = 0.19231 → invert second, EURBRL ≈ 1.08 / 0.19231 ≈ 5.6159
func TestCross_AutoInvertSecond(t *testing.T) {
	a := pricing.Pair{BaseCCY: "EUR", QuoteCCY: "USD", Rate: dec("1.0800")}
	b := pricing.Pair{BaseCCY: "BRL", QuoteCCY: "USD", Rate: dec("0.19231")}
	got, err := pricing.Cross(a, b)
	if err != nil {
		t.Fatalf("Cross: %v", err)
	}
	if got.BaseCCY != "EUR" || got.QuoteCCY != "BRL" {
		t.Fatalf("pair: %s/%s want EUR/BRL", got.BaseCCY, got.QuoteCCY)
	}
	// 1.0800 × (1/0.19231) = 1.0800 × 5.19994800... ≈ 5.6159
	almostEqualP(t, got.Rate, dec("5.6159"), "0.0001")
}

// USDEUR = 0.925, USDBRL = 5.20 → invert first; EURBRL = (1/0.925) × 5.20 ≈ 5.6216
func TestCross_AutoInvertFirst(t *testing.T) {
	a := pricing.Pair{BaseCCY: "USD", QuoteCCY: "EUR", Rate: dec("0.925")}
	b := pricing.Pair{BaseCCY: "USD", QuoteCCY: "BRL", Rate: dec("5.20")}
	got, err := pricing.Cross(a, b)
	if err != nil {
		t.Fatalf("Cross: %v", err)
	}
	if got.BaseCCY != "EUR" || got.QuoteCCY != "BRL" {
		t.Fatalf("pair: %s/%s want EUR/BRL", got.BaseCCY, got.QuoteCCY)
	}
	almostEqualP(t, got.Rate, dec("5.6216"), "0.0001")
}

// GBPUSD = 1.27, JPYUSD = 0.0064 → invert second → GBPJPY = 1.27 / 0.0064 = 198.4375
func TestCross_GBPJPY(t *testing.T) {
	a := pricing.Pair{BaseCCY: "GBP", QuoteCCY: "USD", Rate: dec("1.27")}
	b := pricing.Pair{BaseCCY: "JPY", QuoteCCY: "USD", Rate: dec("0.0064")}
	got, err := pricing.Cross(a, b)
	if err != nil {
		t.Fatalf("Cross: %v", err)
	}
	almostEqualP(t, got.Rate, dec("198.4375"), "0.0001")
}

func TestCross_RejectsIdenticalPair(t *testing.T) {
	p := pricing.Pair{BaseCCY: "EUR", QuoteCCY: "USD", Rate: dec("1.08")}
	_, err := pricing.Cross(p, p)
	if !errors.Is(err, pricing.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestCross_RejectsNoSharedCurrency(t *testing.T) {
	a := pricing.Pair{BaseCCY: "EUR", QuoteCCY: "USD", Rate: dec("1.08")}
	b := pricing.Pair{BaseCCY: "GBP", QuoteCCY: "JPY", Rate: dec("191")}
	_, err := pricing.Cross(a, b)
	if !errors.Is(err, pricing.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestCross_RejectsInvalidPair(t *testing.T) {
	a := pricing.Pair{BaseCCY: "EU", QuoteCCY: "USD", Rate: dec("1.08")}
	b := pricing.Pair{BaseCCY: "USD", QuoteCCY: "BRL", Rate: dec("5.20")}
	_, err := pricing.Cross(a, b)
	if !errors.Is(err, pricing.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestPair_InvertRoundTrip(t *testing.T) {
	p := pricing.Pair{BaseCCY: "EUR", QuoteCCY: "USD", Rate: dec("1.0800")}
	inv, err := p.Invert()
	if err != nil {
		t.Fatalf("Invert: %v", err)
	}
	if inv.BaseCCY != "USD" || inv.QuoteCCY != "EUR" {
		t.Fatalf("swap: %s/%s want USD/EUR", inv.BaseCCY, inv.QuoteCCY)
	}
	back, _ := inv.Invert()
	almostEqualP(t, back.Rate, p.Rate, "0.0000001")
}

package pricing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	refdomain "github.com/revenutech/exchangeos/modules/refdata/domain"
	refpricing "github.com/revenutech/exchangeos/modules/refdata/infrastructure/pricing"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestEngine_HappyPath_FlatSpread(t *testing.T) {
	book := refdomain.NewSpotRateBook(5 * time.Second)
	_ = book.Put(refdomain.SpotRate{
		BaseCCY: "EUR", QuoteCCY: "USD",
		Mid: dec("1.0800"), AsOf: time.Now().UTC(),
	})
	eng := refpricing.New(book, refpricing.FlatSpreadPolicy{Value: dec("0.0001")})

	mid, half, err := eng.GetMidRate(context.Background(), "EUR", "USD")
	if err != nil {
		t.Fatalf("GetMidRate: %v", err)
	}
	if !mid.Equal(dec("1.0800")) {
		t.Fatalf("mid: got %s want 1.0800", mid)
	}
	if !half.Equal(dec("0.0001")) {
		t.Fatalf("half: got %s want 0.0001", half)
	}
}

func TestEngine_PerPairSpread(t *testing.T) {
	book := refdomain.NewSpotRateBook(0) // no staleness
	_ = book.Put(refdomain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: dec("1.08"), AsOf: time.Now().UTC()})
	_ = book.Put(refdomain.SpotRate{BaseCCY: "USD", QuoteCCY: "BRL", Mid: dec("5.10"), AsOf: time.Now().UTC()})

	pol := refpricing.PerPairSpreadPolicy{
		ByPair: map[string]decimal.Decimal{
			"EUR/USD": dec("0.0001"),
			"USD/BRL": dec("0.0050"),
		},
		Default: dec("0.0010"),
	}
	eng := refpricing.New(book, pol)

	_, halfEUR, _ := eng.GetMidRate(context.Background(), "EUR", "USD")
	if !halfEUR.Equal(dec("0.0001")) {
		t.Fatalf("EUR/USD half: %s", halfEUR)
	}
	_, halfBRL, _ := eng.GetMidRate(context.Background(), "USD", "BRL")
	if !halfBRL.Equal(dec("0.0050")) {
		t.Fatalf("USD/BRL half: %s", halfBRL)
	}
	// Unknown pair would fall back to Default, but book has no rate → ErrNotFound.
	_, _, err := eng.GetMidRate(context.Background(), "GBP", "JPY")
	if !errors.Is(err, refdomain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestEngine_StalePropagates(t *testing.T) {
	book := refdomain.NewSpotRateBook(1 * time.Millisecond)
	_ = book.Put(refdomain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: dec("1.08"),
		AsOf: time.Now().UTC().Add(-1 * time.Hour)})
	eng := refpricing.New(book, refpricing.FlatSpreadPolicy{Value: dec("0.0001")})
	_, _, err := eng.GetMidRate(context.Background(), "EUR", "USD")
	if !errors.Is(err, refdomain.ErrStale) {
		t.Fatalf("expected ErrStale, got %v", err)
	}
}

func TestEngine_NilBookErrors(t *testing.T) {
	eng := refpricing.New(nil, refpricing.FlatSpreadPolicy{Value: dec("0.0001")})
	_, _, err := eng.GetMidRate(context.Background(), "EUR", "USD")
	if err == nil {
		t.Fatal("expected error on nil book")
	}
}

func TestEngine_DefaultSpreadWhenPolicyNil(t *testing.T) {
	book := refdomain.NewSpotRateBook(0)
	_ = book.Put(refdomain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: dec("1.08"), AsOf: time.Now().UTC()})
	eng := refpricing.New(book, nil) // nil spread → defaults to flat 0.0002
	_, half, _ := eng.GetMidRate(context.Background(), "EUR", "USD")
	if !half.Equal(dec("0.0002")) {
		t.Fatalf("default half: %s", half)
	}
}

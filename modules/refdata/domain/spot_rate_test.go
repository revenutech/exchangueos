package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/refdata/domain"
)

func TestSpotRateBook_PutLookup(t *testing.T) {
	b := domain.NewSpotRateBook(5 * time.Second)
	if err := b.Put(domain.SpotRate{
		BaseCCY: "eur", QuoteCCY: "USD",
		Mid:  decimal.RequireFromString("1.0800"),
		AsOf: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	r, ok := b.Lookup("EUR", "USD")
	if !ok {
		t.Fatal("not found")
	}
	if r.BaseCCY != "EUR" || r.QuoteCCY != "USD" {
		t.Fatalf("ccy normalisation: %s/%s", r.BaseCCY, r.QuoteCCY)
	}
	if !r.Mid.Equal(decimal.RequireFromString("1.0800")) {
		t.Fatalf("mid: %s", r.Mid)
	}
}

func TestSpotRateBook_LookupFreshFresh(t *testing.T) {
	b := domain.NewSpotRateBook(5 * time.Second)
	now := time.Now().UTC()
	_ = b.Put(domain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.08"), AsOf: now})
	r, fresh, err := b.LookupFresh("EUR", "USD", now)
	if err != nil || !fresh {
		t.Fatalf("expected fresh, got err=%v fresh=%v", err, fresh)
	}
	if !r.Mid.Equal(decimal.RequireFromString("1.08")) {
		t.Fatalf("mid: %s", r.Mid)
	}
}

func TestSpotRateBook_LookupFreshStale(t *testing.T) {
	b := domain.NewSpotRateBook(1 * time.Second)
	past := time.Now().UTC().Add(-1 * time.Hour)
	_ = b.Put(domain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.08"), AsOf: past})
	_, fresh, err := b.LookupFresh("EUR", "USD", time.Now().UTC())
	if !errors.Is(err, domain.ErrStale) || fresh {
		t.Fatalf("expected stale, got err=%v fresh=%v", err, fresh)
	}
}

func TestSpotRateBook_LookupFreshNotFound(t *testing.T) {
	b := domain.NewSpotRateBook(0)
	_, _, err := b.LookupFresh("EUR", "USD", time.Now().UTC())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSpotRateBook_PutRejectsBadInputs(t *testing.T) {
	b := domain.NewSpotRateBook(5 * time.Second)
	cases := []struct {
		name string
		in   domain.SpotRate
	}{
		{"bad base", domain.SpotRate{BaseCCY: "EU", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.08")}},
		{"same ccy", domain.SpotRate{BaseCCY: "USD", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.0")}},
		{"zero mid", domain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: decimal.Zero}},
		{"negative mid", domain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: decimal.RequireFromString("-1")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := b.Put(tc.in); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestSpotRateBook_PutAsOfDefaultsToNow(t *testing.T) {
	b := domain.NewSpotRateBook(0)
	if err := b.Put(domain.SpotRate{BaseCCY: "EUR", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.08")}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	r, _ := b.Lookup("EUR", "USD")
	if r.AsOf.IsZero() {
		t.Fatal("AsOf not defaulted")
	}
}

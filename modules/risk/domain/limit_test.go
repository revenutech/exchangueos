package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/risk/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestNewLimit_Happy(t *testing.T) {
	l, err := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(),
		Type:     domain.LimitCounterparty,
		Scope:    "deutdeff",
		Cap:      dec("10000000"),
		Currency: "usd",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if l.Currency() != "USD" || l.Scope() != "DEUTDEFF" {
		t.Errorf("normalisation: scope=%s ccy=%s", l.Scope(), l.Currency())
	}
	if !l.Available().Equal(dec("10000000")) {
		t.Errorf("available: %s", l.Available())
	}
	if !l.UtilisationPct().IsZero() {
		t.Errorf("pct: %s", l.UtilisationPct())
	}
}

func TestNewLimit_BadInputs(t *testing.T) {
	cases := []struct {
		name string
		in   domain.NewLimitInput
	}{
		{"nil tenant", domain.NewLimitInput{Type: domain.LimitCurrency, Scope: "USD", Cap: dec("1"), Currency: "USD"}},
		{"bad type", domain.NewLimitInput{TenantID: uuid.New(), Type: "BOGUS", Scope: "X", Cap: dec("1"), Currency: "USD"}},
		{"zero cap", domain.NewLimitInput{TenantID: uuid.New(), Type: domain.LimitCurrency, Scope: "USD", Cap: dec("0"), Currency: "USD"}},
		{"bad ccy", domain.NewLimitInput{TenantID: uuid.New(), Type: domain.LimitCurrency, Scope: "USD", Cap: dec("1"), Currency: "DOLLAR"}},
		{"missing scope", domain.NewLimitInput{TenantID: uuid.New(), Type: domain.LimitCounterparty, Cap: dec("1"), Currency: "USD"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewLimit(tc.in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReserve_WithinCap(t *testing.T) {
	l, _ := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(),
		Type:     domain.LimitDV01,
		Cap:      dec("1000"),
		Currency: "USD",
	})
	if err := l.Reserve(dec("300")); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if !l.Utilised().Equal(dec("300")) {
		t.Fatalf("utilised: %s", l.Utilised())
	}
	if !l.Available().Equal(dec("700")) {
		t.Fatalf("available: %s", l.Available())
	}
	if !l.UtilisationPct().Equal(dec("30")) {
		t.Fatalf("pct: %s", l.UtilisationPct())
	}
}

func TestReserve_OverCap_Breached(t *testing.T) {
	l, _ := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(),
		Type:     domain.LimitDV01,
		Cap:      dec("1000"),
		Currency: "USD",
	})
	_ = l.Reserve(dec("900"))
	if err := l.Reserve(dec("200")); !errors.Is(err, domain.ErrBreached) {
		t.Fatalf("want ErrBreached, got %v", err)
	}
	// Original utilised unchanged (no partial reserve).
	if !l.Utilised().Equal(dec("900")) {
		t.Fatalf("utilised after breach: %s", l.Utilised())
	}
}

func TestRelease_ClampsAtZero(t *testing.T) {
	l, _ := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(),
		Type:     domain.LimitVaR,
		Cap:      dec("1000"),
		Currency: "USD",
	})
	_ = l.Reserve(dec("400"))
	_ = l.Release(dec("1000"))
	if !l.Utilised().IsZero() {
		t.Fatalf("expected clamp at zero, got %s", l.Utilised())
	}
}

func TestSetUtilised_RejectsNegative(t *testing.T) {
	l, _ := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(),
		Type:     domain.LimitVaR,
		Cap:      dec("1000"),
		Currency: "USD",
	})
	if err := l.SetUtilised(dec("-1")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestReserveRelease_RejectNonPositive(t *testing.T) {
	l, _ := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(), Type: domain.LimitDV01, Cap: dec("1000"), Currency: "USD",
	})
	if err := l.Reserve(dec("0")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("zero reserve: %v", err)
	}
	if err := l.Release(dec("-1")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("negative release: %v", err)
	}
}

func TestVersion_IncrementsOnMutation(t *testing.T) {
	l, _ := domain.NewLimit(domain.NewLimitInput{
		TenantID: uuid.New(), Type: domain.LimitDV01, Cap: dec("1000"), Currency: "USD",
	})
	v0 := l.Version()
	_ = l.Reserve(dec("1"))
	if l.Version() != v0+1 {
		t.Fatalf("version: got %d want %d", l.Version(), v0+1)
	}
}

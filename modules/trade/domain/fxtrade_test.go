package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/trade/domain"
)

func validInput(t *testing.T) domain.NewTradeInput {
	t.Helper()
	tradeDate := time.Date(2026, 5, 24, 14, 0, 0, 0, time.UTC)
	return domain.NewTradeInput{
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
		DealRate:       decimal.NewFromFloat(1.08),
		TradeDate:      tradeDate,
		ValueDate:      tradeDate.Add(48 * time.Hour),
	}
}

func TestNewFXTrade_HappyPath(t *testing.T) {
	tr, err := domain.NewFXTrade(validInput(t))
	if err != nil {
		t.Fatalf("NewFXTrade: %v", err)
	}
	if tr.Status() != domain.StatusPending {
		t.Errorf("initial status: got %s want PENDING", tr.Status())
	}
	if tr.Version() != 1 {
		t.Errorf("initial version: got %d want 1", tr.Version())
	}
	events := tr.PendingEvents()
	if len(events) != 1 {
		t.Fatalf("pending events: got %d want 1", len(events))
	}
	if events[0].EventName() != "trade.created.v1" {
		t.Errorf("first event: got %s want trade.created.v1", events[0].EventName())
	}
}

func TestRN_FX_001_CurrencyPairMustDiffer(t *testing.T) {
	in := validInput(t)
	in.SoldCurrency = in.BoughtCurrency
	_, err := domain.NewFXTrade(in)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestRN_FX_001_CurrencyMustBeISO4217Alpha3(t *testing.T) {
	for _, bad := range []string{"EU", "EURO", "eur1", "us$"} {
		in := validInput(t)
		in.BoughtCurrency = bad
		_, err := domain.NewFXTrade(in)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("currency %q: want ErrInvalidInput, got %v", bad, err)
		}
	}
}

func TestRN_FX_026_AmountsAndRateMustBePositive(t *testing.T) {
	cases := []struct {
		name  string
		mutate func(in *domain.NewTradeInput)
	}{
		{"bought zero", func(in *domain.NewTradeInput) { in.BoughtAmount = decimal.Zero }},
		{"bought negative", func(in *domain.NewTradeInput) { in.BoughtAmount = decimal.NewFromInt(-1) }},
		{"sold zero", func(in *domain.NewTradeInput) { in.SoldAmount = decimal.Zero }},
		{"rate zero", func(in *domain.NewTradeInput) { in.DealRate = decimal.Zero }},
		{"rate negative", func(in *domain.NewTradeInput) { in.DealRate = decimal.NewFromFloat(-1.0) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput(t)
			tc.mutate(&in)
			_, err := domain.NewFXTrade(in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestBIC_StructuralLength(t *testing.T) {
	for _, bad := range []string{"", "AB", "TOOLONGBIC1234"} {
		in := validInput(t)
		in.BuyerBIC = bad
		_, err := domain.NewFXTrade(in)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("bic %q: want ErrInvalidInput, got %v", bad, err)
		}
	}
}

func TestBuyerSellerCannotBeSame(t *testing.T) {
	in := validInput(t)
	in.SellerBIC = in.BuyerBIC
	_, err := domain.NewFXTrade(in)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestValueDateNotBeforeTradeDate(t *testing.T) {
	in := validInput(t)
	in.ValueDate = in.TradeDate.Add(-24 * time.Hour)
	_, err := domain.NewFXTrade(in)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestLifecycle_ConfirmCancelSettle(t *testing.T) {
	tr, err := domain.NewFXTrade(validInput(t))
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Confirm PENDING → CONFIRMED
	if err := tr.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if tr.Status() != domain.StatusConfirmed {
		t.Fatalf("after Confirm: got %s want CONFIRMED", tr.Status())
	}
	// Idempotent: re-Confirm doesn't error
	if err := tr.Confirm(); err != nil {
		t.Fatalf("re-Confirm: %v", err)
	}

	// CONFIRMED → SETTLING → SETTLED
	if err := tr.MarkSettling(); err != nil {
		t.Fatalf("MarkSettling: %v", err)
	}
	if err := tr.MarkSettled("CLS-REF-9999"); err != nil {
		t.Fatalf("MarkSettled: %v", err)
	}
	if tr.Status() != domain.StatusSettled {
		t.Fatalf("after MarkSettled: got %s want SETTLED", tr.Status())
	}

	// Cannot cancel after settled
	if err := tr.Cancel("late"); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("cancel after settle: want ErrInvalidTransition, got %v", err)
	}
}

func TestCancel_RequiresReason(t *testing.T) {
	tr, _ := domain.NewFXTrade(validInput(t))
	if err := tr.Cancel(""); !errors.Is(err, domain.ErrCancelReasonRequired) {
		t.Fatalf("want ErrCancelReasonRequired, got %v", err)
	}
}

func TestVersion_IncrementsOnTransition(t *testing.T) {
	tr, _ := domain.NewFXTrade(validInput(t))
	v0 := tr.Version()
	if err := tr.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if tr.Version() != v0+1 {
		t.Fatalf("version: got %d want %d", tr.Version(), v0+1)
	}
}

func TestPendingEvents_Lifecycle(t *testing.T) {
	tr, _ := domain.NewFXTrade(validInput(t))
	_ = tr.Confirm()
	_ = tr.MarkSettling()
	_ = tr.MarkSettled("CLS-REF-1")
	events := tr.PendingEvents()
	wantNames := []string{"trade.created.v1", "trade.confirmed.v1", "trade.settling.v1", "trade.settled.v1"}
	if len(events) != len(wantNames) {
		t.Fatalf("events: got %d want %d", len(events), len(wantNames))
	}
	for i, n := range wantNames {
		if events[i].EventName() != n {
			t.Errorf("event[%d]: got %s want %s", i, events[i].EventName(), n)
		}
	}

	tr.MarkEventsCommitted()
	if got := tr.PendingEvents(); len(got) != 0 {
		t.Fatalf("after commit: got %d events want 0", len(got))
	}
}

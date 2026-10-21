package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/trade/application"
	"github.com/revenutech/exchangeos/modules/trade/domain"
	"github.com/revenutech/exchangeos/modules/trade/infrastructure/memory"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newSvc(t *testing.T) (*application.Service, *memory.TradeRepo, *memory.NoopPublisher) {
	t.Helper()
	tr := memory.NewTradeRepo()
	pub := memory.NewNoopPublisher()
	return application.NewService(tr, pub), tr, pub
}

func validReq(t *testing.T) application.BookTradeRequest {
	t.Helper()
	td := time.Date(2026, 5, 22, 14, 0, 0, 0, time.UTC)
	return application.BookTradeRequest{
		TenantID:       uuid.New(),
		ExternalRef:    "REF-001",
		Type:           domain.TradeTypeSpot,
		Venue:          domain.VenueCLS,
		BuyerBIC:       "DEUTDEFF",
		SellerBIC:      "CHASUS33",
		BoughtCurrency: "EUR",
		BoughtAmount:   decimal.NewFromInt(1_000_000),
		SoldCurrency:   "USD",
		SoldAmount:     decimal.NewFromInt(1_080_000),
		DealRate:       dec("1.08"),
		TradeDate:      td,
		ValueDate:      td.Add(48 * time.Hour),
	}
}

func TestBookTrade_PersistsAndPublishes(t *testing.T) {
	svc, repo, pub := newSvc(t)
	tr, err := svc.BookTrade(context.Background(), validReq(t))
	if err != nil {
		t.Fatalf("BookTrade: %v", err)
	}
	if tr.Status() != domain.StatusPending {
		t.Errorf("status: %s", tr.Status())
	}
	if got, _ := repo.Get(context.Background(), tr.ID()); got == nil {
		t.Fatal("not persisted")
	}
	if len(pub.Published) != 1 || pub.Published[0].EventName() != "trade.created.v1" {
		t.Fatalf("publication: %d events", len(pub.Published))
	}
}

func TestBookTrade_PropagatesValidationError(t *testing.T) {
	svc, _, _ := newSvc(t)
	bad := validReq(t)
	bad.BoughtCurrency = bad.SoldCurrency // RN_FX_001 violation
	_, err := svc.BookTrade(context.Background(), bad)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want domain.ErrInvalidInput, got %v", err)
	}
}

func TestGetTrade_BadIDAndMissing(t *testing.T) {
	svc, _, _ := newSvc(t)
	if _, err := svc.GetTrade(context.Background(), uuid.Nil); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil id: %v", err)
	}
	if _, err := svc.GetTrade(context.Background(), uuid.New()); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

func TestLifecycle_ConfirmSettle(t *testing.T) {
	svc, _, pub := newSvc(t)
	ctx := context.Background()
	tr, _ := svc.BookTrade(ctx, validReq(t))

	if _, err := svc.ConfirmTrade(ctx, tr.ID()); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if _, err := svc.MarkSettling(ctx, tr.ID()); err != nil {
		t.Fatalf("MarkSettling: %v", err)
	}
	out, err := svc.MarkSettled(ctx, tr.ID(), "CLS-REF-1")
	if err != nil {
		t.Fatalf("MarkSettled: %v", err)
	}
	if out.Status() != domain.StatusSettled {
		t.Errorf("final: %s", out.Status())
	}
	// Events: created + confirmed + settling + settled = 4
	wantNames := []string{"trade.created.v1", "trade.confirmed.v1", "trade.settling.v1", "trade.settled.v1"}
	if len(pub.Published) != len(wantNames) {
		t.Fatalf("events: got %d want %d", len(pub.Published), len(wantNames))
	}
	for i, n := range wantNames {
		if pub.Published[i].EventName() != n {
			t.Errorf("event[%d]: %s", i, pub.Published[i].EventName())
		}
	}
}

func TestCancel_RequiresReason(t *testing.T) {
	svc, _, _ := newSvc(t)
	tr, _ := svc.BookTrade(context.Background(), validReq(t))
	if _, err := svc.CancelTrade(context.Background(), tr.ID(), ""); !errors.Is(err, domain.ErrCancelReasonRequired) {
		t.Fatalf("want ErrCancelReasonRequired, got %v", err)
	}
}

func TestListTrades_FilterByStatusAndWindow(t *testing.T) {
	svc, _, _ := newSvc(t)
	ctx := context.Background()
	tenant := uuid.New()

	// Two trades on different days under the same tenant; confirm one.
	mkReq := func(d time.Time) application.BookTradeRequest {
		r := validReq(t)
		r.TenantID = tenant
		r.TradeDate = d
		r.ValueDate = d.Add(48 * time.Hour)
		return r
	}
	d1 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	t1, _ := svc.BookTrade(ctx, mkReq(d1))
	t2, _ := svc.BookTrade(ctx, mkReq(d2))
	_, _ = svc.ConfirmTrade(ctx, t2.ID())

	// Confirmed only → expect t2.
	confirmed, err := svc.ListTrades(ctx, tenant, domain.StatusConfirmed, time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Fatalf("List confirmed: %v", err)
	}
	if len(confirmed) != 1 || confirmed[0].ID() != t2.ID() {
		t.Fatalf("confirmed: got %d, IDs %v", len(confirmed), idsOf(confirmed))
	}
	// All trades (no status filter), date window covering only d1.
	all, _ := svc.ListTrades(ctx, tenant, "", d1.Add(-1*time.Hour), d1.Add(1*time.Hour), 100)
	if len(all) != 1 || all[0].ID() != t1.ID() {
		t.Fatalf("date filter: got %d, IDs %v", len(all), idsOf(all))
	}
}

func idsOf(list []*domain.FXTrade) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		out = append(out, t.ID())
	}
	return out
}

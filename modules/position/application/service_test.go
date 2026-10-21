package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/position/application"
	"github.com/revenutech/exchangeos/modules/position/domain"
	"github.com/revenutech/exchangeos/modules/position/infrastructure/memory"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestApplyTradeLeg_CreatesPositionOnMiss(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()

	p, err := svc.ApplyTradeLeg(ctx, tenant, "USD", domain.SideBuy, dec("1000000"))
	if err != nil {
		t.Fatalf("ApplyTradeLeg: %v", err)
	}
	if !p.Long().Equal(dec("1000000")) {
		t.Fatalf("long: %s", p.Long())
	}
	// Subsequent application: same aggregate, not a duplicate.
	p2, _ := svc.ApplyTradeLeg(ctx, tenant, "USD", domain.SideSell, dec("400000"))
	if !p2.Net().Equal(dec("600000")) {
		t.Fatalf("net: %s", p2.Net())
	}
}

func TestList_OrderedByCurrency(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()
	_, _ = svc.ApplyTradeLeg(ctx, tenant, "EUR", domain.SideBuy, dec("1"))
	_, _ = svc.ApplyTradeLeg(ctx, tenant, "USD", domain.SideBuy, dec("1"))
	_, _ = svc.ApplyTradeLeg(ctx, tenant, "GBP", domain.SideBuy, dec("1"))

	list, _ := svc.List(ctx, tenant)
	if len(list) != 3 {
		t.Fatalf("count: %d", len(list))
	}
	got := []string{list[0].Currency(), list[1].Currency(), list[2].Currency()}
	want := []string{"EUR", "GBP", "USD"}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d]: %s vs %s", i, got[i], want[i])
		}
	}
}

func TestGet_BadInputsAndMissing(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	if _, err := svc.Get(context.Background(), uuid.Nil, "USD"); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil tenant: %v", err)
	}
	if _, err := svc.Get(context.Background(), uuid.New(), ""); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("empty ccy: %v", err)
	}
	if _, err := svc.Get(context.Background(), uuid.New(), "USD"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

func TestApplyTradeLeg_BadInput(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	if _, err := svc.ApplyTradeLeg(context.Background(), uuid.New(), "USD", domain.SideBuy, dec("0")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want domain.ErrInvalidInput, got %v", err)
	}
}

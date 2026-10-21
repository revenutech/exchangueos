package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/netreport/application"
	"github.com/revenutech/exchangeos/modules/netreport/domain"
	"github.com/revenutech/exchangeos/modules/netreport/infrastructure/memory"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestGenerateAndGet(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()
	cycle := uuid.New()

	if _, err := svc.Generate(ctx, domain.NewNetReportInput{
		TenantID: tenant, CycleID: cycle, Currency: "USD",
		GrossPayIn: dec("1000000"), GrossPayOut: dec("750000"), TradeCount: 3,
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	n, err := svc.Get(ctx, cycle, "usd")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !n.NetSettlement().Equal(dec("250000")) {
		t.Errorf("net: %s", n.NetSettlement())
	}
}

func TestListByCycle_Ordered(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()
	cycle := uuid.New()
	for _, ccy := range []string{"USD", "EUR", "GBP"} {
		_, _ = svc.Generate(ctx, domain.NewNetReportInput{
			TenantID: tenant, CycleID: cycle, Currency: ccy,
			GrossPayIn: dec("1"), GrossPayOut: dec("1"),
		})
	}
	list, err := svc.ListByCycle(ctx, cycle)
	if err != nil {
		t.Fatalf("ListByCycle: %v", err)
	}
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

func TestGet_BadInputAndMissing(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	if _, err := svc.Get(context.Background(), uuid.Nil, "USD"); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil cycle: %v", err)
	}
	if _, err := svc.Get(context.Background(), uuid.New(), ""); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("empty ccy: %v", err)
	}
	if _, err := svc.Get(context.Background(), uuid.New(), "USD"); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/risk/application"
	"github.com/revenutech/exchangeos/modules/risk/domain"
	"github.com/revenutech/exchangeos/modules/risk/infrastructure/memory"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestCheckLimit_Allowed(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()
	_, _ = svc.CreateLimit(ctx, domain.NewLimitInput{
		TenantID: tenant, Type: domain.LimitCounterparty, Scope: "DEUTDEFF",
		Cap: dec("10000000"), Currency: "USD",
	})

	res, err := svc.CheckLimit(ctx, tenant, domain.LimitCounterparty, "deutdeff", dec("1000000"))
	if err != nil {
		t.Fatalf("CheckLimit: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("expected allowed: %+v", res)
	}
}

func TestCheckLimit_Breached(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()
	l, _ := svc.CreateLimit(ctx, domain.NewLimitInput{
		TenantID: tenant, Type: domain.LimitDV01, Cap: dec("1000"), Currency: "USD",
	})
	_, _ = svc.Reserve(ctx, tenant, domain.LimitDV01, "", dec("900"))

	res, err := svc.CheckLimit(ctx, tenant, domain.LimitDV01, "", dec("200"))
	if err != nil {
		t.Fatalf("CheckLimit: %v", err)
	}
	if res.Allowed {
		t.Fatalf("expected breached: %+v", res)
	}
	if len(res.BreachedLimits) != 1 || res.BreachedLimits[0] != l.ID().String() {
		t.Fatalf("breached list mismatch: %+v", res)
	}
}

func TestReserveRelease_RoundTrip(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	ctx := context.Background()
	tenant := uuid.New()
	_, _ = svc.CreateLimit(ctx, domain.NewLimitInput{
		TenantID: tenant, Type: domain.LimitVaR, Cap: dec("5000"), Currency: "USD",
	})
	l, err := svc.Reserve(ctx, tenant, domain.LimitVaR, "", dec("1000"))
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if !l.Utilised().Equal(dec("1000")) {
		t.Fatalf("util: %s", l.Utilised())
	}
	l, _ = svc.Release(ctx, tenant, domain.LimitVaR, "", dec("400"))
	if !l.Utilised().Equal(dec("600")) {
		t.Fatalf("after release: %s", l.Utilised())
	}
}

func TestCheckLimit_BadInputs(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	if _, err := svc.CheckLimit(context.Background(), uuid.Nil, domain.LimitVaR, "", dec("1")); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil tenant: %v", err)
	}
	if _, err := svc.CheckLimit(context.Background(), uuid.New(), domain.LimitVaR, "", dec("-1")); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("negative exposure: %v", err)
	}
}

func TestCheckLimit_NotFound(t *testing.T) {
	svc := application.NewService(memory.NewRepo())
	if _, err := svc.CheckLimit(context.Background(), uuid.New(), domain.LimitVaR, "", dec("1")); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

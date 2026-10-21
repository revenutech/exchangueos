package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/refdata/application"
	"github.com/revenutech/exchangeos/modules/refdata/domain"
	"github.com/revenutech/exchangeos/modules/refdata/infrastructure/memory"
)

func newSvc(t *testing.T) (*application.Service, *memory.CurrencyRepo, *memory.CalendarRepo, *memory.BICRepo, *memory.SSIRepo) {
	t.Helper()
	cr := memory.NewCurrencyRepo()
	cal := memory.NewCalendarRepo()
	br := memory.NewBICRepo()
	sr := memory.NewSSIRepo()
	return application.NewService(cr, cal, br, sr), cr, cal, br, sr
}

func TestListCurrencies_OrderedAndFiltered(t *testing.T) {
	svc, cr, _, _, _ := newSvc(t)
	ctx := context.Background()

	usd, _ := domain.NewCurrency("USD", "Dollar", 2, true, false)
	eur, _ := domain.NewCurrency("EUR", "Euro", 2, true, false)
	xxx, _ := domain.NewCurrency("XXX", "Inactive", 2, false, false)
	xxx.Deactivate()

	cr.Put(usd)
	cr.Put(eur)
	cr.Put(xxx)

	all, err := svc.ListCurrencies(ctx, false)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 || all[0].Code() != "EUR" || all[1].Code() != "USD" || all[2].Code() != "XXX" {
		t.Fatalf("ordering / completeness: %v", all)
	}

	active, _ := svc.ListCurrencies(ctx, true)
	if len(active) != 2 {
		t.Fatalf("active filter: want 2, got %d", len(active))
	}
}

func TestGetCurrency_NormalisesAndNotFound(t *testing.T) {
	svc, cr, _, _, _ := newSvc(t)
	ctx := context.Background()

	usd, _ := domain.NewCurrency("USD", "Dollar", 2, true, false)
	cr.Put(usd)

	got, err := svc.GetCurrency(ctx, "  usd ")
	if err != nil {
		t.Fatalf("GetCurrency: %v", err)
	}
	if got.Code() != "USD" {
		t.Fatalf("code: got %s want USD", got.Code())
	}

	if _, err := svc.GetCurrency(ctx, "ZZZ"); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := svc.GetCurrency(ctx, "BAD"); err == nil {
		t.Fatal("bad-length code: expected error")
	}
}

func TestGetCalendar(t *testing.T) {
	svc, _, cal, _, _ := newSvc(t)
	ctx := context.Background()

	c, _ := domain.NewCalendar("USD_NYC", []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	cal.Put(c)

	got, err := svc.GetCalendar(ctx, "usd_nyc")
	if err != nil {
		t.Fatalf("GetCalendar: %v", err)
	}
	if got.ID() != "USD_NYC" {
		t.Fatalf("id: got %s want USD_NYC", got.ID())
	}
	if !got.IsHoliday(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("holiday missing")
	}

	if _, err := svc.GetCalendar(ctx, ""); !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("empty id: want ErrInvalidInput, got %v", err)
	}
	if _, err := svc.GetCalendar(ctx, "NOPE"); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("not-found: want ErrNotFound, got %v", err)
	}
}

func TestResolveBIC(t *testing.T) {
	svc, _, _, br, _ := newSvc(t)
	ctx := context.Background()

	b, _ := domain.NewBICRecord("DEUTDEFF", "Deutsche Bank AG", "DE", "")
	br.Put(b)

	got, err := svc.ResolveBIC(ctx, "deutdeff")
	if err != nil {
		t.Fatalf("ResolveBIC: %v", err)
	}
	if got.BIC() != "DEUTDEFF" {
		t.Fatalf("bic: got %s want DEUTDEFF", got.BIC())
	}

	if _, err := svc.ResolveBIC(ctx, "SHORT"); !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("bad length: want ErrInvalidInput, got %v", err)
	}
	if _, err := svc.ResolveBIC(ctx, "UNKNOWN1"); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("not found: want ErrNotFound, got %v", err)
	}
}

func TestGetSSI_PicksActiveByTime(t *testing.T) {
	svc, _, _, _, sr := newSvc(t)
	ctx := context.Background()
	tenant := uuid.New()
	now := time.Now().UTC()

	older, _ := domain.NewSSI(domain.NewSSIInput{
		TenantID: tenant, CounterpartyBIC: "CHASUS33", Currency: "USD",
		BeneficiaryBIC: "CHASUS33", AccountNumber: "OLD-ACC",
		ValidFrom: now.AddDate(-2, 0, 0), ValidTo: now.AddDate(-1, 0, 0),
	})
	current, _ := domain.NewSSI(domain.NewSSIInput{
		TenantID: tenant, CounterpartyBIC: "CHASUS33", Currency: "USD",
		BeneficiaryBIC: "CHASUS33", AccountNumber: "CURRENT-ACC",
		ValidFrom: now.AddDate(0, -1, 0), ValidTo: now.AddDate(1, 0, 0),
	})
	sr.Put(older)
	sr.Put(current)

	got, err := svc.GetSSI(ctx, tenant, "chasus33", "usd", now)
	if err != nil {
		t.Fatalf("GetSSI: %v", err)
	}
	if got.AccountNumber() != "CURRENT-ACC" {
		t.Fatalf("picked stale SSI: %s", got.AccountNumber())
	}

	// Time before validFrom of any SSI → not found.
	if _, err := svc.GetSSI(ctx, tenant, "CHASUS33", "USD", now.AddDate(-5, 0, 0)); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("ancient time: want ErrNotFound, got %v", err)
	}
}

func TestGetSSI_BadInputs(t *testing.T) {
	svc, _, _, _, _ := newSvc(t)
	ctx := context.Background()

	if _, err := svc.GetSSI(ctx, uuid.Nil, "CHASUS33", "USD", time.Time{}); !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("nil tenant: want ErrInvalidInput, got %v", err)
	}
	if _, err := svc.GetSSI(ctx, uuid.New(), "SHORT", "USD", time.Time{}); !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("bad bic: want ErrInvalidInput, got %v", err)
	}
	if _, err := svc.GetSSI(ctx, uuid.New(), "CHASUS33", "DOLLAR", time.Time{}); !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("bad ccy: want ErrInvalidInput, got %v", err)
	}
}

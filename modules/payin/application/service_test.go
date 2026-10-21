package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/payin/application"
	"github.com/revenutech/exchangeos/modules/payin/domain"
	"github.com/revenutech/exchangeos/modules/payin/infrastructure/memory"
)

func newSvc(t *testing.T) (*application.Service, *memory.Repo, *memory.NoopPublisher) {
	t.Helper()
	r := memory.NewRepo()
	pub := memory.NewNoopPublisher()
	return application.NewService(r, pub), r, pub
}

func validIn(t *testing.T, deadline time.Time) domain.NewPayInInput {
	return domain.NewPayInInput{
		TenantID: uuid.New(),
		CycleID:  uuid.New(),
		Currency: "USD",
		Amount:   decimal.NewFromInt(1_000_000),
		Band:     domain.BandPIN3,
		Deadline: deadline,
	}
}

func TestCreateSubmitConfirm_HappyPath(t *testing.T) {
	svc, _, pub := newSvc(t)
	ctx := context.Background()
	dl := time.Now().UTC().Add(1 * time.Hour)

	p, err := svc.Create(ctx, validIn(t, dl))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Submit(ctx, p.ID(), time.Now().UTC()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := svc.Confirm(ctx, p.ID(), time.Now().UTC()); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	wantNames := []string{"payin.created.v1", "payin.submitted.v1", "payin.confirmed.v1"}
	if len(pub.Published) != len(wantNames) {
		t.Fatalf("events: %d", len(pub.Published))
	}
	for i, n := range wantNames {
		if pub.Published[i].EventName() != n {
			t.Errorf("event[%d]: %s", i, pub.Published[i].EventName())
		}
	}
}

func TestSubmitAfterDeadline_PropagatesMissed(t *testing.T) {
	svc, _, _ := newSvc(t)
	ctx := context.Background()
	dl := time.Now().UTC().Add(-1 * time.Hour)
	p, _ := svc.Create(ctx, validIn(t, dl))
	if _, err := svc.Submit(ctx, p.ID(), time.Now().UTC()); !errors.Is(err, domain.ErrDeadlineMissed) {
		t.Fatalf("want ErrDeadlineMissed, got %v", err)
	}
}

func TestListByCycle_FiltersAndOrders(t *testing.T) {
	svc, _, _ := newSvc(t)
	ctx := context.Background()
	cycle := uuid.New()
	mk := func(ccy string, hours int) domain.NewPayInInput {
		in := validIn(t, time.Now().UTC().Add(time.Duration(hours)*time.Hour))
		in.CycleID = cycle
		in.Currency = ccy
		return in
	}
	_, _ = svc.Create(ctx, mk("EUR", 2))
	_, _ = svc.Create(ctx, mk("USD", 4))
	_, _ = svc.Create(ctx, mk("USD", 1)) // earlier deadline first
	// Different cycle — should be excluded.
	otherCycle := validIn(t, time.Now().UTC().Add(1*time.Hour))
	_, _ = svc.Create(ctx, otherCycle)

	list, err := svc.ListByCycle(ctx, cycle)
	if err != nil {
		t.Fatalf("ListByCycle: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("count: got %d want 3", len(list))
	}
	// EUR first (alphabetical), then USD ordered by deadline ascending.
	gotCurs := []string{list[0].Currency(), list[1].Currency(), list[2].Currency()}
	wantCurs := []string{"EUR", "USD", "USD"}
	for i := range gotCurs {
		if gotCurs[i] != wantCurs[i] {
			t.Errorf("[%d]: %s vs %s", i, gotCurs[i], wantCurs[i])
		}
	}
}

func TestGet_BadIdAndMissing(t *testing.T) {
	svc, _, _ := newSvc(t)
	if _, err := svc.Get(context.Background(), uuid.Nil); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil: %v", err)
	}
	if _, err := svc.Get(context.Background(), uuid.New()); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

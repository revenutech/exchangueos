package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cls_settlement/application"
	"github.com/revenutech/exchangeos/modules/cls_settlement/domain"
	"github.com/revenutech/exchangeos/modules/cls_settlement/infrastructure/memory"
)

func newSvc(t *testing.T) (*application.Service, *memory.CycleRepo, *memory.NoopPublisher) {
	t.Helper()
	repo := memory.NewCycleRepo()
	pub := memory.NewNoopPublisher()
	return application.NewService(repo, pub), repo, pub
}

func TestOpenCycle_HappyAndConflict(t *testing.T) {
	svc, _, pub := newSvc(t)
	ctx := context.Background()
	tenant := uuid.New()
	day := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

	c, err := svc.OpenCycle(ctx, tenant, day)
	if err != nil {
		t.Fatalf("OpenCycle: %v", err)
	}
	if c.Status() != domain.StatusOpen {
		t.Errorf("status: %s", c.Status())
	}
	if len(pub.Published) != 1 || pub.Published[0].EventName() != "cls_cycle.opened.v1" {
		t.Fatalf("publication: %d events", len(pub.Published))
	}

	// Same date → conflict.
	_, err = svc.OpenCycle(ctx, tenant, day)
	if !errors.Is(err, application.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestOpenCycle_BadInputs(t *testing.T) {
	svc, _, _ := newSvc(t)
	day := time.Now().UTC()
	if _, err := svc.OpenCycle(context.Background(), uuid.Nil, day); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil tenant: %v", err)
	}
	if _, err := svc.OpenCycle(context.Background(), uuid.New(), time.Time{}); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("zero date: %v", err)
	}
}

func TestLifecycle_FullHappy(t *testing.T) {
	svc, _, pub := newSvc(t)
	ctx := context.Background()
	tenant := uuid.New()
	day := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	c, _ := svc.OpenCycle(ctx, tenant, day)

	if _, err := svc.AttachTrade(ctx, c.ID(), uuid.New()); err != nil {
		t.Fatalf("AttachTrade: %v", err)
	}
	now := time.Now().UTC()
	if _, err := svc.EnterPayInWindow(ctx, c.ID(), now); err != nil {
		t.Fatalf("EnterPayInWindow: %v", err)
	}
	if _, err := svc.EnterSettling(ctx, c.ID(), now); err != nil {
		t.Fatalf("EnterSettling: %v", err)
	}
	closed, err := svc.CloseCycle(ctx, c.ID(), now)
	if err != nil {
		t.Fatalf("CloseCycle: %v", err)
	}
	if closed.Status() != domain.StatusClosed {
		t.Fatalf("status: %s", closed.Status())
	}

	wantNames := []string{
		"cls_cycle.opened.v1",
		"cls_cycle.trade_attached.v1",
		"cls_cycle.payin_opened.v1",
		"cls_cycle.settling.v1",
		"cls_cycle.closed.v1",
	}
	if len(pub.Published) != len(wantNames) {
		t.Fatalf("events: got %d want %d", len(pub.Published), len(wantNames))
	}
	for i, n := range wantNames {
		if pub.Published[i].EventName() != n {
			t.Errorf("event[%d]: %s", i, pub.Published[i].EventName())
		}
	}
}

func TestFailCycle_PropagatesDomainError(t *testing.T) {
	svc, _, _ := newSvc(t)
	ctx := context.Background()
	c, _ := svc.OpenCycle(ctx, uuid.New(), time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC))
	if _, err := svc.FailCycle(ctx, c.ID(), time.Now(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput (empty reason), got %v", err)
	}
	if _, err := svc.FailCycle(ctx, c.ID(), time.Now(), "PIN1 missed"); err != nil {
		t.Fatalf("FailCycle: %v", err)
	}
}

func TestGetCycle_BadIdAndMissing(t *testing.T) {
	svc, _, _ := newSvc(t)
	if _, err := svc.GetCycle(context.Background(), uuid.Nil); !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("nil id: %v", err)
	}
	if _, err := svc.GetCycle(context.Background(), uuid.New()); !errors.Is(err, application.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cls_settlement/domain"
)

func TestOpenCycle_AnchorDeadlinesToCET(t *testing.T) {
	c, err := domain.OpenCycle(domain.OpenCycleInput{
		TenantID:  uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 14, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("OpenCycle: %v", err)
	}
	if c.Status() != domain.StatusOpen {
		t.Errorf("status: got %s", c.Status())
	}
	wantDate := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	if !c.CycleDate().Equal(wantDate) {
		t.Errorf("cycle_date normalised: got %v want %v", c.CycleDate(), wantDate)
	}
	// CET = UTC+1 in winter, UTC+2 in summer. May 22 is CEST → UTC+2.
	// 08:00 CEST = 06:00 UTC; 12:00 CEST = 10:00 UTC.
	pin1, _ := c.DeadlineFor("PIN1")
	if !pin1.Equal(time.Date(2026, 5, 22, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("PIN1: got %v want 2026-05-22 06:00 UTC", pin1)
	}
	close := c.ScheduledClose()
	if !close.Equal(time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("scheduled close: got %v want 2026-05-22 10:00 UTC", close)
	}
}

func TestOpenCycle_RejectsBadInput(t *testing.T) {
	_, err := domain.OpenCycle(domain.OpenCycleInput{CycleDate: time.Now()})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("nil tenant: want ErrInvalidInput, got %v", err)
	}
	_, err = domain.OpenCycle(domain.OpenCycleInput{TenantID: uuid.New()})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("zero date: want ErrInvalidInput, got %v", err)
	}
}

func TestCycle_AttachTrade_Idempotent(t *testing.T) {
	c, _ := domain.OpenCycle(domain.OpenCycleInput{
		TenantID: uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	tid := uuid.New()
	if err := c.AttachTrade(tid); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := c.AttachTrade(tid); err != nil {
		t.Fatalf("re-attach: %v", err)
	}
	if len(c.TradeIDs()) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(c.TradeIDs()))
	}
}

func TestCycle_AttachTrade_RejectsAfterSettling(t *testing.T) {
	c, _ := domain.OpenCycle(domain.OpenCycleInput{
		TenantID: uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	_ = c.EnterPayInWindow(time.Now())
	_ = c.EnterSettling(time.Now())
	err := c.AttachTrade(uuid.New())
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestCycle_FullLifecycle_HappyPath(t *testing.T) {
	c, _ := domain.OpenCycle(domain.OpenCycleInput{
		TenantID: uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	_ = c.AttachTrade(uuid.New())
	if err := c.EnterPayInWindow(time.Now().UTC()); err != nil {
		t.Fatalf("EnterPayInWindow: %v", err)
	}
	if err := c.EnterSettling(time.Now().UTC()); err != nil {
		t.Fatalf("EnterSettling: %v", err)
	}
	if err := c.Close(time.Now().UTC()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if c.Status() != domain.StatusClosed {
		t.Fatalf("final: %s", c.Status())
	}
	wantNames := []string{
		"cls_cycle.opened.v1",
		"cls_cycle.trade_attached.v1",
		"cls_cycle.payin_opened.v1",
		"cls_cycle.settling.v1",
		"cls_cycle.closed.v1",
	}
	events := c.PendingEvents()
	if len(events) != len(wantNames) {
		t.Fatalf("events: got %d want %d", len(events), len(wantNames))
	}
	for i, n := range wantNames {
		if events[i].EventName() != n {
			t.Errorf("event[%d]: got %s want %s", i, events[i].EventName(), n)
		}
	}
}

func TestCycle_Fail_TerminalThenForbids(t *testing.T) {
	c, _ := domain.OpenCycle(domain.OpenCycleInput{
		TenantID: uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	if err := c.Fail(time.Now(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing reason: want ErrInvalidInput, got %v", err)
	}
	if err := c.Fail(time.Now(), "PIN1 missed"); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if c.Status() != domain.StatusFailed {
		t.Fatalf("status: %s", c.Status())
	}
	if err := c.Fail(time.Now(), "again"); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("re-fail: want ErrInvalidTransition, got %v", err)
	}
}

func TestCycle_VersionIncrements(t *testing.T) {
	c, _ := domain.OpenCycle(domain.OpenCycleInput{
		TenantID: uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	v0 := c.Version()
	_ = c.EnterPayInWindow(time.Now())
	if c.Version() != v0+1 {
		t.Fatalf("version: got %d want %d", c.Version(), v0+1)
	}
}

func TestDeadlineFor_UnknownBand(t *testing.T) {
	c, _ := domain.OpenCycle(domain.OpenCycleInput{
		TenantID: uuid.New(),
		CycleDate: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
	})
	_, err := c.DeadlineFor("BOGUS")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

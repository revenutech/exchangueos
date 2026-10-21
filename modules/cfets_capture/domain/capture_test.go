package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cfets_capture/domain"
)

func validIn(t *testing.T) domain.NewCaptureInput {
	return domain.NewCaptureInput{
		TenantID:     uuid.New(),
		TradeID:      uuid.New(),
		SubmitterRef: "REF-001",
	}
}

func TestNewCapture_Happy(t *testing.T) {
	c, err := domain.NewCapture(validIn(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Status() != domain.StatusDraft {
		t.Errorf("status: %s", c.Status())
	}
	if got := len(c.PendingEvents()); got != 1 {
		t.Errorf("events: %d", got)
	}
}

func TestNewCapture_BadInputs(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*domain.NewCaptureInput)
	}{
		{"nil tenant", func(in *domain.NewCaptureInput) { in.TenantID = uuid.Nil }},
		{"nil trade", func(in *domain.NewCaptureInput) { in.TradeID = uuid.Nil }},
		{"empty ref", func(in *domain.NewCaptureInput) { in.SubmitterRef = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validIn(t)
			tc.mutate(&in)
			_, err := domain.NewCapture(in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestLifecycle_HappyAck(t *testing.T) {
	c, _ := domain.NewCapture(validIn(t))
	if err := c.Submit(time.Now().UTC()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := c.Ack(time.Now().UTC(), "CFETS-DEAL-001"); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	if c.CFETSDealID() != "CFETS-DEAL-001" {
		t.Errorf("dealID: %s", c.CFETSDealID())
	}
	if err := c.NotifyCounterparty(time.Now().UTC()); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if c.Status() != domain.StatusNotified {
		t.Errorf("final: %s", c.Status())
	}
	wantNames := []string{
		"cfets_capture.drafted.v1",
		"cfets_capture.submitted.v1",
		"cfets_capture.acked.v1",
		"cfets_capture.notified.v1",
	}
	events := c.PendingEvents()
	if len(events) != len(wantNames) {
		t.Fatalf("count: %d", len(events))
	}
	for i, n := range wantNames {
		if events[i].EventName() != n {
			t.Errorf("[%d]: %s", i, events[i].EventName())
		}
	}
}

func TestLifecycle_RejectPath(t *testing.T) {
	c, _ := domain.NewCapture(validIn(t))
	_ = c.Submit(time.Now().UTC())
	if err := c.Reject(time.Now().UTC(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing reason: want ErrInvalidInput, got %v", err)
	}
	if err := c.Reject(time.Now().UTC(), "currency pair not allowed"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if c.Status() != domain.StatusRejected {
		t.Errorf("status: %s", c.Status())
	}
	// Subsequent transitions forbidden.
	if err := c.Ack(time.Now().UTC(), "X"); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("ack-after-reject: want ErrInvalidTransition, got %v", err)
	}
}

func TestAck_RequiresDealID(t *testing.T) {
	c, _ := domain.NewCapture(validIn(t))
	_ = c.Submit(time.Now().UTC())
	if err := c.Ack(time.Now().UTC(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestSubmit_RequiresDraft(t *testing.T) {
	c, _ := domain.NewCapture(validIn(t))
	_ = c.Submit(time.Now().UTC())
	if err := c.Submit(time.Now().UTC()); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

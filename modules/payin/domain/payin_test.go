package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/payin/domain"
)

func validInput(t *testing.T, deadline time.Time) domain.NewPayInInput {
	t.Helper()
	return domain.NewPayInInput{
		TenantID: uuid.New(),
		CycleID:  uuid.New(),
		Currency: "usd",
		Amount:   decimal.NewFromInt(1_000_000),
		Band:     domain.BandPIN3,
		Deadline: deadline,
	}
}

func TestNewPayInInstruction_Happy(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	p, err := domain.NewPayInInstruction(validInput(t, dl))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Status() != domain.StatusPending {
		t.Errorf("status: %s", p.Status())
	}
	if p.Currency() != "USD" {
		t.Errorf("ccy normalised: %s", p.Currency())
	}
	if got := len(p.PendingEvents()); got != 1 {
		t.Errorf("events: %d", got)
	}
}

func TestNewPayInInstruction_BadInputs(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	cases := []struct {
		name   string
		mutate func(*domain.NewPayInInput)
	}{
		{"nil tenant", func(in *domain.NewPayInInput) { in.TenantID = uuid.Nil }},
		{"nil cycle", func(in *domain.NewPayInInput) { in.CycleID = uuid.Nil }},
		{"bad ccy", func(in *domain.NewPayInInput) { in.Currency = "DOLLAR" }},
		{"zero amount", func(in *domain.NewPayInInput) { in.Amount = decimal.Zero }},
		{"bad band", func(in *domain.NewPayInInput) { in.Band = "PIN9" }},
		{"zero deadline", func(in *domain.NewPayInInput) { in.Deadline = time.Time{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput(t, dl)
			tc.mutate(&in)
			_, err := domain.NewPayInInstruction(in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestPayIn_Submit_BeforeDeadline(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	p, _ := domain.NewPayInInstruction(validInput(t, dl))
	if err := p.Submit(time.Now().UTC()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if p.Status() != domain.StatusSubmitted {
		t.Errorf("after submit: %s", p.Status())
	}
}

func TestPayIn_Submit_AfterDeadline_AutoFails(t *testing.T) {
	dl := time.Now().UTC().Add(-1 * time.Hour)
	p, _ := domain.NewPayInInstruction(validInput(t, dl))
	err := p.Submit(time.Now().UTC())
	if !errors.Is(err, domain.ErrDeadlineMissed) {
		t.Fatalf("want ErrDeadlineMissed, got %v", err)
	}
	if p.Status() != domain.StatusFailed {
		t.Errorf("after missed: %s", p.Status())
	}
}

func TestPayIn_ConfirmHappy(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	p, _ := domain.NewPayInInstruction(validInput(t, dl))
	_ = p.Submit(time.Now().UTC())
	if err := p.Confirm(time.Now().UTC()); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if p.Status() != domain.StatusConfirmed {
		t.Errorf("status: %s", p.Status())
	}
}

func TestPayIn_ConfirmBeforeSubmit_Rejected(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	p, _ := domain.NewPayInInstruction(validInput(t, dl))
	if err := p.Confirm(time.Now().UTC()); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestPayIn_FailRequiresReason(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	p, _ := domain.NewPayInInstruction(validInput(t, dl))
	if err := p.Fail(time.Now(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestPayIn_FullEventTrail(t *testing.T) {
	dl := time.Now().UTC().Add(1 * time.Hour)
	p, _ := domain.NewPayInInstruction(validInput(t, dl))
	_ = p.Submit(time.Now().UTC())
	_ = p.Confirm(time.Now().UTC())
	want := []string{"payin.created.v1", "payin.submitted.v1", "payin.confirmed.v1"}
	events := p.PendingEvents()
	if len(events) != len(want) {
		t.Fatalf("count: got %d want %d", len(events), len(want))
	}
	for i, n := range want {
		if events[i].EventName() != n {
			t.Errorf("event[%d]: %s", i, events[i].EventName())
		}
	}
}

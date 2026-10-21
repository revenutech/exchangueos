package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cfets_confirmation/domain"
)

func validIn(t *testing.T) domain.NewConfirmationInput {
	return domain.NewConfirmationInput{
		TenantID:    uuid.New(),
		TradeID:     uuid.New(),
		CFETSDealID: "CFETS-DEAL-001",
	}
}

func TestNewConfirmation_Happy(t *testing.T) {
	c, err := domain.NewConfirmation(validIn(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Status() != domain.StatusConfirming {
		t.Errorf("status: %s", c.Status())
	}
	if len(c.PendingEvents()) != 1 {
		t.Errorf("events: %d", len(c.PendingEvents()))
	}
}

func TestNewConfirmation_BadInputs(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*domain.NewConfirmationInput)
	}{
		{"nil tenant", func(in *domain.NewConfirmationInput) { in.TenantID = uuid.Nil }},
		{"nil trade", func(in *domain.NewConfirmationInput) { in.TradeID = uuid.Nil }},
		{"empty deal", func(in *domain.NewConfirmationInput) { in.CFETSDealID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validIn(t)
			tc.mutate(&in)
			_, err := domain.NewConfirmation(in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestPaired_HappyPath(t *testing.T) {
	c, _ := domain.NewConfirmation(validIn(t))
	if err := c.MarkPaired(time.Now().UTC()); err != nil {
		t.Fatalf("MarkPaired: %v", err)
	}
	if c.Status() != domain.StatusConfirmed {
		t.Errorf("status: %s", c.Status())
	}
}

func TestUnpairedThenPaired(t *testing.T) {
	c, _ := domain.NewConfirmation(validIn(t))
	if err := c.MarkUnpaired(time.Now().UTC()); err != nil {
		t.Fatalf("MarkUnpaired: %v", err)
	}
	if c.Status() != domain.StatusUnpaired {
		t.Errorf("status: %s", c.Status())
	}
	if err := c.MarkPaired(time.Now().UTC()); err != nil {
		t.Fatalf("MarkPaired from UNPAIRED: %v", err)
	}
	if c.Status() != domain.StatusConfirmed {
		t.Errorf("status: %s", c.Status())
	}
}

func TestRejected_FromUnpaired(t *testing.T) {
	c, _ := domain.NewConfirmation(validIn(t))
	_ = c.MarkUnpaired(time.Now().UTC())
	if err := c.MarkRejected(time.Now().UTC(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing reason: want ErrInvalidInput, got %v", err)
	}
	if err := c.MarkRejected(time.Now().UTC(), "counterparty disagreed"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if c.Status() != domain.StatusRejected {
		t.Errorf("status: %s", c.Status())
	}
	// After REJECTED no further transition.
	if err := c.MarkPaired(time.Now().UTC()); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("after-rejected: want ErrInvalidTransition, got %v", err)
	}
}

func TestPaired_AfterConfirmed_Rejected(t *testing.T) {
	c, _ := domain.NewConfirmation(validIn(t))
	_ = c.MarkPaired(time.Now().UTC())
	if err := c.MarkPaired(time.Now().UTC()); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("re-pair: want ErrInvalidTransition, got %v", err)
	}
}

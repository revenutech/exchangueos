package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PayInStatus is the lifecycle state of a single PayIn instruction.
type PayInStatus string

const (
	StatusPending   PayInStatus = "PENDING"
	StatusSubmitted PayInStatus = "SUBMITTED"
	StatusConfirmed PayInStatus = "CONFIRMED"
	StatusFailed    PayInStatus = "FAILED"
)

// DeadlineBand identifies the CLS PayIn window. Mirror of netting_cutoffs.band.
type DeadlineBand string

const (
	BandPIN1 DeadlineBand = "PIN1"
	BandPIN2 DeadlineBand = "PIN2"
	BandPIN3 DeadlineBand = "PIN3"
)

// PayInInstruction is the aggregate root for one currency obligation in a CLS cycle.
type PayInInstruction struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	cycleID       uuid.UUID
	currency      string
	amount        decimal.Decimal
	band          DeadlineBand
	deadline      time.Time
	status        PayInStatus
	submittedAt   time.Time
	confirmedAt   time.Time
	failureReason string
	version       int
	events        []DomainEvent
}

// NewPayInInput parameterises construction.
type NewPayInInput struct {
	TenantID uuid.UUID
	CycleID  uuid.UUID
	Currency string
	Amount   decimal.Decimal
	Band     DeadlineBand
	Deadline time.Time
}

// NewPayInInstruction validates and constructs an instruction in PENDING state.
func NewPayInInstruction(in NewPayInInput) (*PayInInstruction, error) {
	if in.TenantID == uuid.Nil || in.CycleID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id + cycle_id required", ErrInvalidInput)
	}
	ccy := strings.ToUpper(strings.TrimSpace(in.Currency))
	if len(ccy) != 3 {
		return nil, fmt.Errorf("%w: currency must be ISO 4217 alpha-3", ErrInvalidInput)
	}
	if !in.Amount.IsPositive() {
		return nil, fmt.Errorf("%w: amount must be > 0", ErrInvalidInput)
	}
	switch in.Band {
	case BandPIN1, BandPIN2, BandPIN3:
	default:
		return nil, fmt.Errorf("%w: band must be PIN1|PIN2|PIN3, got %q", ErrInvalidInput, in.Band)
	}
	if in.Deadline.IsZero() {
		return nil, fmt.Errorf("%w: deadline required", ErrInvalidInput)
	}
	id := uuid.New()
	p := &PayInInstruction{
		id:       id,
		tenantID: in.TenantID,
		cycleID:  in.CycleID,
		currency: ccy,
		amount:   in.Amount,
		band:     in.Band,
		deadline: in.Deadline.UTC(),
		status:   StatusPending,
		version:  1,
	}
	p.recordEvent(EventPayInCreated{
		InstructionID: id,
		TenantID:      in.TenantID,
		CycleID:       in.CycleID,
		Currency:      ccy,
		Amount:        in.Amount,
		Band:          in.Band,
		Deadline:      p.deadline,
		OccurredAt:    time.Now().UTC(),
	})
	return p, nil
}

// ─── Accessors ─────────────────────────────────────────────────────────────

func (p *PayInInstruction) ID() uuid.UUID           { return p.id }
func (p *PayInInstruction) TenantID() uuid.UUID     { return p.tenantID }
func (p *PayInInstruction) CycleID() uuid.UUID      { return p.cycleID }
func (p *PayInInstruction) Currency() string        { return p.currency }
func (p *PayInInstruction) Amount() decimal.Decimal { return p.amount }
func (p *PayInInstruction) Band() DeadlineBand      { return p.band }
func (p *PayInInstruction) Deadline() time.Time     { return p.deadline }
func (p *PayInInstruction) Status() PayInStatus     { return p.status }
func (p *PayInInstruction) SubmittedAt() time.Time  { return p.submittedAt }
func (p *PayInInstruction) ConfirmedAt() time.Time  { return p.confirmedAt }
func (p *PayInInstruction) FailureReason() string   { return p.failureReason }
func (p *PayInInstruction) Version() int            { return p.version }
func (p *PayInInstruction) PendingEvents() []DomainEvent {
	return append([]DomainEvent(nil), p.events...)
}
func (p *PayInInstruction) MarkEventsCommitted() { p.events = nil }

// ─── Lifecycle ─────────────────────────────────────────────────────────────

// Submit transitions PENDING → SUBMITTED if `at` is before the deadline.
func (p *PayInInstruction) Submit(at time.Time) error {
	if p.status != StatusPending {
		return fmt.Errorf("%w: submit requires PENDING, got %s", ErrInvalidTransition, p.status)
	}
	at = at.UTC()
	if at.After(p.deadline) {
		// Auto-fail when missed deadline.
		_ = p.Fail(at, "deadline missed at submit")
		return ErrDeadlineMissed
	}
	p.status = StatusSubmitted
	p.submittedAt = at
	p.version++
	p.recordEvent(EventPayInSubmitted{InstructionID: p.id, At: at})
	return nil
}

// Confirm transitions SUBMITTED → CONFIRMED (CLS has cleared the payment).
func (p *PayInInstruction) Confirm(at time.Time) error {
	if p.status != StatusSubmitted {
		return fmt.Errorf("%w: confirm requires SUBMITTED, got %s", ErrInvalidTransition, p.status)
	}
	p.status = StatusConfirmed
	p.confirmedAt = at.UTC()
	p.version++
	p.recordEvent(EventPayInConfirmed{InstructionID: p.id, At: p.confirmedAt})
	return nil
}

// Fail moves any non-terminal state to FAILED with a reason.
func (p *PayInInstruction) Fail(at time.Time, reason string) error {
	switch p.status {
	case StatusConfirmed, StatusFailed:
		return fmt.Errorf("%w: cannot fail from %s", ErrInvalidTransition, p.status)
	}
	if reason == "" {
		return fmt.Errorf("%w: reason required", ErrInvalidInput)
	}
	p.status = StatusFailed
	p.failureReason = reason
	p.version++
	p.recordEvent(EventPayInFailed{InstructionID: p.id, Reason: reason, At: at.UTC()})
	return nil
}

func (p *PayInInstruction) recordEvent(e DomainEvent) { p.events = append(p.events, e) }

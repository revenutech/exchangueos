package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CaptureStatus is the lifecycle state of a CFETS capture.
type CaptureStatus string

const (
	StatusDraft     CaptureStatus = "DRAFT"
	StatusSubmitted CaptureStatus = "SUBMITTED"
	StatusAck       CaptureStatus = "ACK"
	StatusRejected  CaptureStatus = "REJECTED"
	StatusNotified  CaptureStatus = "NOTIFIED"
)

// CFETSCapture is the aggregate root for one CFETS Trade Capture submission.
type CFETSCapture struct {
	id              uuid.UUID
	tenantID        uuid.UUID
	tradeID         uuid.UUID // local Trade aggregate id
	submitterRef    string
	cfetsDealID     string // assigned on ACK
	status          CaptureStatus
	submittedAt     time.Time
	ackAt           time.Time
	rejectionReason string
	notifiedAt      time.Time
	version         int
	events          []DomainEvent
}

// NewCaptureInput parameterises construction.
type NewCaptureInput struct {
	TenantID     uuid.UUID
	TradeID      uuid.UUID
	SubmitterRef string
}

// NewCapture constructs a capture in DRAFT state.
func NewCapture(in NewCaptureInput) (*CFETSCapture, error) {
	if in.TenantID == uuid.Nil || in.TradeID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id + trade_id required", ErrInvalidInput)
	}
	if in.SubmitterRef == "" {
		return nil, fmt.Errorf("%w: submitter_ref required", ErrInvalidInput)
	}
	id := uuid.New()
	c := &CFETSCapture{
		id:           id,
		tenantID:     in.TenantID,
		tradeID:      in.TradeID,
		submitterRef: in.SubmitterRef,
		status:       StatusDraft,
		version:      1,
	}
	c.recordEvent(EventCaptureDrafted{
		CaptureID:    id,
		TenantID:     in.TenantID,
		TradeID:      in.TradeID,
		SubmitterRef: in.SubmitterRef,
		OccurredAt:   time.Now().UTC(),
	})
	return c, nil
}

// ── Accessors ──────────────────────────────────────────────────────────────
func (c *CFETSCapture) ID() uuid.UUID           { return c.id }
func (c *CFETSCapture) TenantID() uuid.UUID     { return c.tenantID }
func (c *CFETSCapture) TradeID() uuid.UUID      { return c.tradeID }
func (c *CFETSCapture) SubmitterRef() string    { return c.submitterRef }
func (c *CFETSCapture) CFETSDealID() string     { return c.cfetsDealID }
func (c *CFETSCapture) Status() CaptureStatus   { return c.status }
func (c *CFETSCapture) SubmittedAt() time.Time  { return c.submittedAt }
func (c *CFETSCapture) AckAt() time.Time        { return c.ackAt }
func (c *CFETSCapture) NotifiedAt() time.Time   { return c.notifiedAt }
func (c *CFETSCapture) RejectionReason() string { return c.rejectionReason }
func (c *CFETSCapture) Version() int            { return c.version }
func (c *CFETSCapture) PendingEvents() []DomainEvent {
	return append([]DomainEvent(nil), c.events...)
}
func (c *CFETSCapture) MarkEventsCommitted() { c.events = nil }

// ── Lifecycle ──────────────────────────────────────────────────────────────

// Submit transitions DRAFT → SUBMITTED (member sent fxtr.031 to CFETS).
func (c *CFETSCapture) Submit(at time.Time) error {
	if c.status != StatusDraft {
		return fmt.Errorf("%w: submit requires DRAFT, got %s", ErrInvalidTransition, c.status)
	}
	c.status = StatusSubmitted
	c.submittedAt = at.UTC()
	c.version++
	c.recordEvent(EventCaptureSubmitted{CaptureID: c.id, At: c.submittedAt})
	return nil
}

// Ack transitions SUBMITTED → ACK with the assigned CFETS deal id.
func (c *CFETSCapture) Ack(at time.Time, cfetsDealID string) error {
	if c.status != StatusSubmitted {
		return fmt.Errorf("%w: ack requires SUBMITTED, got %s", ErrInvalidTransition, c.status)
	}
	if cfetsDealID == "" {
		return fmt.Errorf("%w: cfets_deal_id required", ErrInvalidInput)
	}
	c.status = StatusAck
	c.ackAt = at.UTC()
	c.cfetsDealID = cfetsDealID
	c.version++
	c.recordEvent(EventCaptureAcked{CaptureID: c.id, At: c.ackAt, CFETSDealID: cfetsDealID})
	return nil
}

// Reject transitions SUBMITTED → REJECTED with a reason.
func (c *CFETSCapture) Reject(at time.Time, reason string) error {
	if c.status != StatusSubmitted {
		return fmt.Errorf("%w: reject requires SUBMITTED, got %s", ErrInvalidTransition, c.status)
	}
	if reason == "" {
		return fmt.Errorf("%w: rejection reason required", ErrInvalidInput)
	}
	c.status = StatusRejected
	c.rejectionReason = reason
	c.version++
	c.recordEvent(EventCaptureRejected{CaptureID: c.id, At: at.UTC(), Reason: reason})
	return nil
}

// NotifyCounterparty transitions ACK → NOTIFIED (informational; CFETS forwarded fxtr.033).
func (c *CFETSCapture) NotifyCounterparty(at time.Time) error {
	if c.status != StatusAck {
		return fmt.Errorf("%w: notify requires ACK, got %s", ErrInvalidTransition, c.status)
	}
	c.status = StatusNotified
	c.notifiedAt = at.UTC()
	c.version++
	c.recordEvent(EventCaptureNotified{CaptureID: c.id, At: c.notifiedAt})
	return nil
}

func (c *CFETSCapture) recordEvent(e DomainEvent) { c.events = append(c.events, e) }

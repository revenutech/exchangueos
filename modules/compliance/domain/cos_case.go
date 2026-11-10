package domain

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// COSStatus is the lifecycle state of a COSCase.
type COSStatus string

const (
	COSDraft       COSStatus = "DRAFT"
	COSUnderReview COSStatus = "UNDER_REVIEW"
	COSApproved    COSStatus = "APPROVED"
	COSSubmitted   COSStatus = "SUBMITTED"
	COSAccepted    COSStatus = "ACCEPTED"
	COSRejected    COSStatus = "REJECTED"
)

// COSRequiredApprovers is the 4-eyes quorum: two distinct approvers, neither of them the case creator.
const COSRequiredApprovers = 2

var (
	// ErrSelfApproval — the case creator cannot approve their own case (4-eyes).
	ErrSelfApproval = fmt.Errorf("%w: creator cannot approve own COS case", ErrInvalidInput)
	// ErrDuplicateApprover — the same approver counts once.
	ErrDuplicateApprover = fmt.Errorf("%w: approver already recorded", ErrInvalidInput)
)

// COSCase — a Comunicação de Operação Suspeita on its way to SISCOAF (PLD/FT).
//
// RN_FX_039: the COS goes to SISCOAF within 1 business day after the decision. The decision is the second
// approval (APPROVED); DecidedAt is the instant the deadline counts from. The deadline check itself lives in the
// application layer, which owns the business-day calendar.
//
// Lifecycle: DRAFT → UNDER_REVIEW → APPROVED → SUBMITTED → ACCEPTED | REJECTED.
//
// PII: the narrative (and SISCOAF's rejection reason, which may quote it) describes a person. String, GoString,
// LogValue and MarshalJSON never print them, only identifiers, the status and lengths. They use value receivers on
// purpose, so a dereferenced copy is redacted as well. Events carry no narrative.
//
// Other aggregates (counterparty, screening result) are referenced by ID only.
//
// Identities: createdBy and every approver must be the authenticated principal's `sub` claim, never a typed name.
// 4-eyes compares the trimmed text, so two spellings of one person would count as two approvers.
type COSCase struct {
	id                uuid.UUID
	tenantID          uuid.UUID
	counterpartyID    uuid.UUID
	screeningResultID uuid.UUID // optional (uuid.Nil): a case may be opened without a screening hit
	createdBy         string
	narrative         string
	status            COSStatus
	approvers         []string
	createdAt         time.Time
	decidedAt         time.Time
	submittedAt       time.Time
	respondedAt       time.Time
	protocolID        string
	rejectionReason   string
	version           int
	events            []DomainEvent
}

// NewCOSCaseInput parameterises construction.
type NewCOSCaseInput struct {
	TenantID          uuid.UUID
	CounterpartyID    uuid.UUID
	ScreeningResultID uuid.UUID // optional
	CreatedBy         string    // principal that opened the case
	Narrative         string    // PII: the suspicion description
	At                time.Time
}

// NewCOSCase constructs a case in DRAFT state.
func NewCOSCase(in NewCOSCaseInput) (*COSCase, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	if in.CounterpartyID == uuid.Nil {
		return nil, fmt.Errorf("%w: counterparty_id required", ErrInvalidInput)
	}
	creator := strings.TrimSpace(in.CreatedBy)
	if creator == "" {
		return nil, fmt.Errorf("%w: created_by required", ErrInvalidInput)
	}
	if strings.TrimSpace(in.Narrative) == "" {
		return nil, fmt.Errorf("%w: narrative required", ErrInvalidInput)
	}
	if in.At.IsZero() {
		return nil, fmt.Errorf("%w: created_at required", ErrInvalidInput)
	}
	c := &COSCase{
		id:                uuid.New(),
		tenantID:          in.TenantID,
		counterpartyID:    in.CounterpartyID,
		screeningResultID: in.ScreeningResultID,
		createdBy:         creator,
		narrative:         in.Narrative,
		status:            COSDraft,
		createdAt:         in.At.UTC(),
		version:           1,
	}
	c.recordEvent(EventCOSCaseDrafted{
		CaseID: c.id, Tenant: c.tenantID, CounterpartyID: c.counterpartyID,
		ScreeningResultID: c.screeningResultID, CreatedBy: creator, OccurredAt: c.createdAt,
	})
	return c, nil
}

// Accessors
func (c *COSCase) ID() uuid.UUID                { return c.id }
func (c *COSCase) TenantID() uuid.UUID          { return c.tenantID }
func (c *COSCase) CounterpartyID() uuid.UUID    { return c.counterpartyID }
func (c *COSCase) ScreeningResultID() uuid.UUID { return c.screeningResultID }
func (c *COSCase) CreatedBy() string            { return c.createdBy }
func (c *COSCase) Narrative() string            { return c.narrative }
func (c *COSCase) Status() COSStatus            { return c.status }
func (c *COSCase) Approvers() []string          { return slices.Clone(c.approvers) }
func (c *COSCase) CreatedAt() time.Time         { return c.createdAt }
func (c *COSCase) DecidedAt() time.Time         { return c.decidedAt }
func (c *COSCase) SubmittedAt() time.Time       { return c.submittedAt }
func (c *COSCase) RespondedAt() time.Time       { return c.respondedAt }
func (c *COSCase) ProtocolID() string           { return c.protocolID }
func (c *COSCase) RejectionReason() string      { return c.rejectionReason }
func (c *COSCase) Version() int                 { return c.version }

// PendingEvents returns a copy of the events not yet flushed to the outbox.
func (c *COSCase) PendingEvents() []DomainEvent { return append([]DomainEvent(nil), c.events...) }

// MarkEventsCommitted clears the pending events after the outbox write.
func (c *COSCase) MarkEventsCommitted() { c.events = nil }

// ── Lifecycle ──────────────────────────────────────────────────────────────

// SubmitForReview transitions DRAFT → UNDER_REVIEW.
func (c *COSCase) SubmitForReview(at time.Time) error {
	if err := requireTime(at, "review"); err != nil {
		return err
	}
	if c.status != COSDraft {
		return fmt.Errorf("%w: review requires DRAFT, got %s", ErrInvalidTransition, c.status)
	}
	c.status = COSUnderReview
	c.version++
	c.recordEvent(EventCOSCaseReviewRequested{CaseID: c.id, Tenant: c.tenantID, At: at.UTC()})
	return nil
}

// Approve records one approval while UNDER_REVIEW. The second distinct approver, neither of them the creator,
// moves the case to APPROVED and fixes DecidedAt (RN_FX_039).
func (c *COSCase) Approve(approver string, at time.Time) error {
	if err := requireTime(at, "approve"); err != nil {
		return err
	}
	if c.status != COSUnderReview {
		return fmt.Errorf("%w: approve requires UNDER_REVIEW, got %s", ErrInvalidTransition, c.status)
	}
	who := strings.TrimSpace(approver)
	switch {
	case who == "":
		return fmt.Errorf("%w: approver required", ErrInvalidInput)
	case who == c.createdBy:
		return ErrSelfApproval
	case slices.Contains(c.approvers, who):
		return ErrDuplicateApprover
	}
	c.approvers = append(c.approvers, who)
	c.version++
	c.recordEvent(EventCOSCaseApprovalRecorded{CaseID: c.id, Tenant: c.tenantID, ApproverID: who, Count: len(c.approvers), At: at.UTC()})
	if len(c.approvers) >= COSRequiredApprovers {
		c.status = COSApproved
		c.decidedAt = at.UTC()
		c.recordEvent(EventCOSCaseApproved{CaseID: c.id, Tenant: c.tenantID, Approvers: slices.Clone(c.approvers), DecidedAt: c.decidedAt})
	}
	return nil
}

// MarkSubmitted transitions APPROVED → SUBMITTED with the SISCOAF protocol.
func (c *COSCase) MarkSubmitted(at time.Time, protocolID string) error {
	if err := requireTime(at, "submit"); err != nil {
		return err
	}
	if c.status != COSApproved {
		return fmt.Errorf("%w: submit requires APPROVED, got %s", ErrInvalidTransition, c.status)
	}
	if strings.TrimSpace(protocolID) == "" {
		return fmt.Errorf("%w: protocol_id required", ErrInvalidInput)
	}
	c.status = COSSubmitted
	c.submittedAt = at.UTC()
	c.protocolID = protocolID
	c.version++
	c.recordEvent(EventCOSCaseSubmitted{CaseID: c.id, Tenant: c.tenantID, ProtocolID: protocolID, At: c.submittedAt})
	return nil
}

// MarkAccepted transitions SUBMITTED → ACCEPTED.
func (c *COSCase) MarkAccepted(at time.Time) error {
	if err := requireTime(at, "accept"); err != nil {
		return err
	}
	if c.status != COSSubmitted {
		return fmt.Errorf("%w: accept requires SUBMITTED, got %s", ErrInvalidTransition, c.status)
	}
	c.status = COSAccepted
	c.respondedAt = at.UTC()
	c.version++
	c.recordEvent(EventCOSCaseAccepted{CaseID: c.id, Tenant: c.tenantID, At: c.respondedAt})
	return nil
}

// MarkRejected transitions SUBMITTED → REJECTED. The reason may quote the narrative: it is PII and stays off the event.
func (c *COSCase) MarkRejected(at time.Time, reason string) error {
	if err := requireTime(at, "reject"); err != nil {
		return err
	}
	if c.status != COSSubmitted {
		return fmt.Errorf("%w: reject requires SUBMITTED, got %s", ErrInvalidTransition, c.status)
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: rejection reason required", ErrInvalidInput)
	}
	c.status = COSRejected
	c.rejectionReason = reason
	c.respondedAt = at.UTC()
	c.version++
	c.recordEvent(EventCOSCaseRejected{CaseID: c.id, Tenant: c.tenantID, At: c.respondedAt})
	return nil
}

func (c *COSCase) recordEvent(e DomainEvent) { c.events = append(c.events, e) }

// requireTime refuses a zero instant: RN_FX_039 counts from DecidedAt, and the audit trail needs every timestamp.
func requireTime(at time.Time, op string) error {
	if at.IsZero() {
		return fmt.Errorf("%w: %s requires a non-zero time", ErrInvalidInput, op)
	}
	return nil
}

// ── Redaction (PII) ────────────────────────────────────────────────────────

func (c COSCase) redacted() string {
	return fmt.Sprintf("COSCase{id=%s tenant=%s status=%s approvals=%d/%d narrative_len=%d rejection_len=%d version=%d}",
		c.id, c.tenantID, c.status, len(c.approvers), COSRequiredApprovers, len(c.narrative), len(c.rejectionReason), c.version)
}

// String never prints the narrative nor the rejection reason.
func (c COSCase) String() string { return c.redacted() }

// GoString covers %#v.
func (c COSCase) GoString() string { return c.redacted() }

// LogValue covers slog.
func (c COSCase) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", c.id.String()),
		slog.String("tenant_id", c.tenantID.String()),
		slog.String("status", string(c.status)),
		slog.Int("approvals", len(c.approvers)),
		slog.Int("narrative_len", len(c.narrative)),
		slog.Int("rejection_len", len(c.rejectionReason)),
		slog.Int("version", c.version),
	)
}

// MarshalJSON is the redacted view (ids, status, approvals, narrative_len, rejection_len, version); persistence goes through the repository, not JSON.
func (c COSCase) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           uuid.UUID `json:"id"`
		TenantID     uuid.UUID `json:"tenant_id"`
		Status       COSStatus `json:"status"`
		Approvals    int       `json:"approvals"`
		NarrativeLen int       `json:"narrative_len"`
		RejectionLen int       `json:"rejection_len"`
		Version      int       `json:"version"`
	}{c.id, c.tenantID, c.status, len(c.approvers), len(c.narrative), len(c.rejectionReason), c.version})
}

// ── Events ─────────────────────────────────────────────────────────────────

// DomainEvent is what aggregates of this module record for the outbox.
type DomainEvent interface {
	EventName() string
	When() time.Time
	TenantID() uuid.UUID
}

// EventCOSCaseDrafted — case opened. No narrative (PII).
type EventCOSCaseDrafted struct {
	CaseID            uuid.UUID
	Tenant            uuid.UUID
	CounterpartyID    uuid.UUID
	ScreeningResultID uuid.UUID
	CreatedBy         string
	OccurredAt        time.Time
}

func (e EventCOSCaseDrafted) EventName() string   { return "compliance.cos_case.drafted.v1" }
func (e EventCOSCaseDrafted) When() time.Time     { return e.OccurredAt }
func (e EventCOSCaseDrafted) TenantID() uuid.UUID { return e.Tenant }

// EventCOSCaseReviewRequested — DRAFT → UNDER_REVIEW.
type EventCOSCaseReviewRequested struct {
	CaseID uuid.UUID
	Tenant uuid.UUID
	At     time.Time
}

func (e EventCOSCaseReviewRequested) EventName() string {
	return "compliance.cos_case.review_requested.v1"
}
func (e EventCOSCaseReviewRequested) When() time.Time     { return e.At }
func (e EventCOSCaseReviewRequested) TenantID() uuid.UUID { return e.Tenant }

// EventCOSCaseApprovalRecorded — one 4-eyes approval.
type EventCOSCaseApprovalRecorded struct {
	CaseID     uuid.UUID
	Tenant     uuid.UUID
	ApproverID string
	Count      int
	At         time.Time
}

func (e EventCOSCaseApprovalRecorded) EventName() string {
	return "compliance.cos_case.approval_recorded.v1"
}
func (e EventCOSCaseApprovalRecorded) When() time.Time     { return e.At }
func (e EventCOSCaseApprovalRecorded) TenantID() uuid.UUID { return e.Tenant }

// EventCOSCaseApproved — quorum reached; RN_FX_039 counts from DecidedAt.
type EventCOSCaseApproved struct {
	CaseID    uuid.UUID
	Tenant    uuid.UUID
	Approvers []string
	DecidedAt time.Time
}

func (e EventCOSCaseApproved) EventName() string   { return "compliance.cos_case.approved.v1" }
func (e EventCOSCaseApproved) When() time.Time     { return e.DecidedAt }
func (e EventCOSCaseApproved) TenantID() uuid.UUID { return e.Tenant }

// EventCOSCaseSubmitted — sent to SISCOAF.
type EventCOSCaseSubmitted struct {
	CaseID     uuid.UUID
	Tenant     uuid.UUID
	ProtocolID string
	At         time.Time
}

func (e EventCOSCaseSubmitted) EventName() string   { return "compliance.cos_case.submitted.v1" }
func (e EventCOSCaseSubmitted) When() time.Time     { return e.At }
func (e EventCOSCaseSubmitted) TenantID() uuid.UUID { return e.Tenant }

// EventCOSCaseAccepted — SISCOAF accepted.
type EventCOSCaseAccepted struct {
	CaseID uuid.UUID
	Tenant uuid.UUID
	At     time.Time
}

func (e EventCOSCaseAccepted) EventName() string   { return "compliance.cos_case.accepted.v1" }
func (e EventCOSCaseAccepted) When() time.Time     { return e.At }
func (e EventCOSCaseAccepted) TenantID() uuid.UUID { return e.Tenant }

// EventCOSCaseRejected — SISCOAF rejected. The reason stays off the event (PII).
type EventCOSCaseRejected struct {
	CaseID uuid.UUID
	Tenant uuid.UUID
	At     time.Time
}

func (e EventCOSCaseRejected) EventName() string   { return "compliance.cos_case.rejected.v1" }
func (e EventCOSCaseRejected) When() time.Time     { return e.At }
func (e EventCOSCaseRejected) TenantID() uuid.UUID { return e.Tenant }

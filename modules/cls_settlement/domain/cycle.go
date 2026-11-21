package domain

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// CycleStatus is the lifecycle state of a CLS cycle.
type CycleStatus string

const (
	StatusOpen        CycleStatus = "OPEN"
	StatusPayInWindow CycleStatus = "PAY_IN_WINDOW"
	StatusSettling    CycleStatus = "SETTLING"
	StatusClosed      CycleStatus = "CLOSED"
	StatusFailed      CycleStatus = "FAILED"
)

// CETLocation is the canonical Europe/Zurich location (CET / CEST).
// All CLS cycle deadlines are expressed in this zone.
var CETLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		// Defensive: fall back to fixed +01:00 when tzdata is missing (e.g. distroless).
		loc = time.FixedZone("CET-fallback", 60*60)
	}
	return loc
}()

// CLSCycle is the aggregate root for one CLS settlement day.
type CLSCycle struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	cycleDate time.Time // date-only UTC

	status         CycleStatus
	openedAt       time.Time
	pin1Deadline   time.Time
	pin2Deadline   time.Time
	pin3Deadline   time.Time
	scheduledClose time.Time
	closedAt       time.Time
	failureReason  string

	tradeIDs []uuid.UUID

	version int
	events  []DomainEvent
}

// OpenCycleInput parameterises OpenCycle.
type OpenCycleInput struct {
	TenantID  uuid.UUID
	CycleDate time.Time // any timestamp on the business date — normalised to date-only UTC
}

// OpenCycle constructs a fresh CLSCycle in OPEN status with the four standard
// deadlines anchored to 07/08/09/10/12 CET on the given business date.
func OpenCycle(in OpenCycleInput) (*CLSCycle, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	if in.CycleDate.IsZero() {
		return nil, fmt.Errorf("%w: cycle_date required", ErrInvalidInput)
	}
	bd := time.Date(in.CycleDate.Year(), in.CycleDate.Month(), in.CycleDate.Day(),
		0, 0, 0, 0, time.UTC)

	deadline := func(hour int) time.Time {
		// Deadline is `hour:00` in CET on the business date.
		t := time.Date(bd.Year(), bd.Month(), bd.Day(), hour, 0, 0, 0, CETLocation)
		return t.UTC()
	}

	id := uuid.New()
	c := &CLSCycle{
		id:             id,
		tenantID:       in.TenantID,
		cycleDate:      bd,
		status:         StatusOpen,
		openedAt:       deadline(7),
		pin1Deadline:   deadline(8),
		pin2Deadline:   deadline(9),
		pin3Deadline:   deadline(10),
		scheduledClose: deadline(12),
		version:        1,
	}
	c.recordEvent(EventCycleOpened{
		CycleID:        id,
		TenantID:       in.TenantID,
		CycleDate:      bd,
		OpenedAt:       c.openedAt,
		ScheduledClose: c.scheduledClose,
	})
	return c, nil
}

// ─── Accessors ─────────────────────────────────────────────────────────────

func (c *CLSCycle) ID() uuid.UUID             { return c.id }
func (c *CLSCycle) TenantID() uuid.UUID       { return c.tenantID }
func (c *CLSCycle) CycleDate() time.Time      { return c.cycleDate }
func (c *CLSCycle) Status() CycleStatus       { return c.status }
func (c *CLSCycle) OpenedAt() time.Time       { return c.openedAt }
func (c *CLSCycle) ClosedAt() time.Time       { return c.closedAt }
func (c *CLSCycle) ScheduledClose() time.Time { return c.scheduledClose }
func (c *CLSCycle) FailureReason() string     { return c.failureReason }
func (c *CLSCycle) Version() int              { return c.version }
func (c *CLSCycle) TradeIDs() []uuid.UUID     { return append([]uuid.UUID(nil), c.tradeIDs...) }
func (c *CLSCycle) PendingEvents() []DomainEvent {
	return append([]DomainEvent(nil), c.events...)
}
func (c *CLSCycle) MarkEventsCommitted() { c.events = nil }

// DeadlineFor returns the CET deadline for a given band.
// band ∈ {"PIN1","PIN2","PIN3"}. Returns zero time + error for unknown band.
func (c *CLSCycle) DeadlineFor(band string) (time.Time, error) {
	switch band {
	case "PIN1":
		return c.pin1Deadline, nil
	case "PIN2":
		return c.pin2Deadline, nil
	case "PIN3":
		return c.pin3Deadline, nil
	default:
		return time.Time{}, fmt.Errorf("%w: unknown band %q", ErrInvalidInput, band)
	}
}

// ─── Lifecycle transitions ─────────────────────────────────────────────────

// AttachTrade enrolls a trade into the cycle. Forbidden once the cycle has
// moved past PAY_IN_WINDOW.
func (c *CLSCycle) AttachTrade(tradeID uuid.UUID) error {
	if tradeID == uuid.Nil {
		return fmt.Errorf("%w: trade_id required", ErrInvalidInput)
	}
	switch c.status {
	case StatusOpen, StatusPayInWindow:
	default:
		return fmt.Errorf("%w: cannot attach trade from %s", ErrInvalidTransition, c.status)
	}
	// Idempotence: if the trade is already attached, no-op.
	for _, id := range c.tradeIDs {
		if id == tradeID {
			return nil
		}
	}
	c.tradeIDs = append(c.tradeIDs, tradeID)
	sort.Slice(c.tradeIDs, func(i, j int) bool {
		return c.tradeIDs[i].String() < c.tradeIDs[j].String()
	})
	c.version++
	c.recordEvent(EventCycleTradeAttached{CycleID: c.id, TradeID: tradeID, OccurredAt: time.Now().UTC()})
	return nil
}

// EnterPayInWindow transitions OPEN → PAY_IN_WINDOW (typically driven by the
// scheduler at the PIN1 deadline).
func (c *CLSCycle) EnterPayInWindow(at time.Time) error {
	if c.status != StatusOpen {
		return fmt.Errorf("%w: enter pay-in requires OPEN, got %s", ErrInvalidTransition, c.status)
	}
	c.status = StatusPayInWindow
	c.version++
	c.recordEvent(EventCyclePayInOpened{CycleID: c.id, At: at.UTC()})
	return nil
}

// EnterSettling transitions PAY_IN_WINDOW → SETTLING (typically driven by the
// scheduler at the PIN3 deadline).
func (c *CLSCycle) EnterSettling(at time.Time) error {
	if c.status != StatusPayInWindow {
		return fmt.Errorf("%w: enter settling requires PAY_IN_WINDOW, got %s", ErrInvalidTransition, c.status)
	}
	c.status = StatusSettling
	c.version++
	c.recordEvent(EventCycleSettling{CycleID: c.id, At: at.UTC()})
	return nil
}

// Close transitions SETTLING → CLOSED with the actual close timestamp.
func (c *CLSCycle) Close(at time.Time) error {
	if c.status != StatusSettling {
		return fmt.Errorf("%w: close requires SETTLING, got %s", ErrInvalidTransition, c.status)
	}
	c.status = StatusClosed
	c.closedAt = at.UTC()
	c.version++
	c.recordEvent(EventCycleClosed{CycleID: c.id, ClosedAt: c.closedAt})
	return nil
}

// Fail moves the cycle to FAILED from any non-terminal state and records a reason.
func (c *CLSCycle) Fail(at time.Time, reason string) error {
	switch c.status {
	case StatusClosed, StatusFailed:
		return fmt.Errorf("%w: cannot fail from %s", ErrInvalidTransition, c.status)
	}
	if reason == "" {
		return fmt.Errorf("%w: failure reason required", ErrInvalidInput)
	}
	c.status = StatusFailed
	c.failureReason = reason
	c.closedAt = at.UTC()
	c.version++
	c.recordEvent(EventCycleFailed{CycleID: c.id, Reason: reason, At: c.closedAt})
	return nil
}

func (c *CLSCycle) recordEvent(e DomainEvent) { c.events = append(c.events, e) }

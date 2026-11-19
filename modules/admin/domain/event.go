package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EventCode mirrors the canonical CLS admi.004 enumeration plus internal codes.
type EventCode string

const (
	EventStartup      EventCode = "STARTUP"
	EventShutdown     EventCode = "SHUTDOWN"
	EventDegraded     EventCode = "DEGRADED"
	EventRecovered    EventCode = "RECOVERED"
	EventCycleOpen    EventCode = "CYCLE_OPEN"
	EventCycleClose   EventCode = "CYCLE_CLOSE"
	EventEODStarted   EventCode = "EOD_STARTED"
	EventEODCompleted EventCode = "EOD_COMPLETED"
)

// SystemEvent — operational event for audit + downstream listeners.
type SystemEvent struct {
	id          uuid.UUID
	code        EventCode
	component   string
	description string
	at          time.Time
	iso20022Ref string // optional admi.x message id correlation
}

// NewSystemEventInput parameterises construction.
type NewSystemEventInput struct {
	Code        EventCode
	Component   string
	Description string
	At          time.Time
	ISO20022Ref string
}

// NewSystemEvent validates and constructs.
func NewSystemEvent(in NewSystemEventInput) (*SystemEvent, error) {
	if !isValidEventCode(in.Code) {
		return nil, fmt.Errorf("%w: code %q", ErrInvalidInput, in.Code)
	}
	if strings.TrimSpace(in.Component) == "" {
		return nil, fmt.Errorf("%w: component required", ErrInvalidInput)
	}
	if in.At.IsZero() {
		in.At = time.Now().UTC()
	}
	return &SystemEvent{
		id:          uuid.New(),
		code:        in.Code,
		component:   in.Component,
		description: in.Description,
		at:          in.At.UTC(),
		iso20022Ref: in.ISO20022Ref,
	}, nil
}

func (e *SystemEvent) ID() uuid.UUID       { return e.id }
func (e *SystemEvent) Code() EventCode     { return e.code }
func (e *SystemEvent) Component() string   { return e.component }
func (e *SystemEvent) Description() string { return e.description }
func (e *SystemEvent) At() time.Time       { return e.at }
func (e *SystemEvent) ISO20022Ref() string { return e.iso20022Ref }

func isValidEventCode(c EventCode) bool {
	switch c {
	case EventStartup, EventShutdown, EventDegraded, EventRecovered,
		EventCycleOpen, EventCycleClose, EventEODStarted, EventEODCompleted:
		return true
	}
	return false
}

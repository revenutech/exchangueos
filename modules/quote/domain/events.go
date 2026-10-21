package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DomainEvent interface {
	EventName() string
	When() time.Time
}

type EventQuoteCreated struct {
	QuoteID      uuid.UUID
	TenantID     uuid.UUID
	RequesterBIC string
	ProviderBIC  string
	Side         string
	BaseCCY      string
	QuoteCCY     string
	Bid          decimal.Decimal
	Ask          decimal.Decimal
	OccurredAt   time.Time
}

func (e EventQuoteCreated) EventName() string { return "quote.created.v1" }
func (e EventQuoteCreated) When() time.Time   { return e.OccurredAt }

type EventQuoteAccepted struct {
	QuoteID    uuid.UUID
	Actor      string
	AcceptedAt time.Time
}

func (e EventQuoteAccepted) EventName() string { return "quote.accepted.v1" }
func (e EventQuoteAccepted) When() time.Time   { return e.AcceptedAt }

type EventRFQRequested struct {
	RFQID      uuid.UUID
	TenantID   uuid.UUID
	BaseCCY    string
	QuoteCCY   string
	Requester  string
	OccurredAt time.Time
}

func (e EventRFQRequested) EventName() string { return "rfq.requested.v1" }
func (e EventRFQRequested) When() time.Time   { return e.OccurredAt }

type EventRFQQuoted struct {
	RFQID      uuid.UUID
	QuoteID    uuid.UUID
	OccurredAt time.Time
}

func (e EventRFQQuoted) EventName() string { return "rfq.quoted.v1" }
func (e EventRFQQuoted) When() time.Time   { return e.OccurredAt }

type EventRFQAccepted struct {
	RFQID      uuid.UUID
	QuoteID    uuid.UUID
	Actor      string
	OccurredAt time.Time
}

func (e EventRFQAccepted) EventName() string { return "rfq.accepted.v1" }
func (e EventRFQAccepted) When() time.Time   { return e.OccurredAt }

type EventRFQRejected struct {
	RFQID      uuid.UUID
	Reason     string
	OccurredAt time.Time
}

func (e EventRFQRejected) EventName() string { return "rfq.rejected.v1" }
func (e EventRFQRejected) When() time.Time   { return e.OccurredAt }

type EventRFQExpired struct {
	RFQID      uuid.UUID
	OccurredAt time.Time
}

func (e EventRFQExpired) EventName() string { return "rfq.expired.v1" }
func (e EventRFQExpired) When() time.Time   { return e.OccurredAt }

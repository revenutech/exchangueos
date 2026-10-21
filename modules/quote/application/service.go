// Package application — Quote/RFQ orchestration over the domain layer.
//
// The PricingEngine interface decouples the service from pkg/pricing (which
// implementations satisfy via a thin adapter). Repositories are interfaces;
// concrete impls live under modules/quote/infrastructure/.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/quote/domain"
)

// ─── Repository + collaborator interfaces ──────────────────────────────────

type QuoteRepository interface {
	Save(ctx context.Context, q *domain.Quote) error
	Get(ctx context.Context, id uuid.UUID) (*domain.Quote, error)
}

type RFQRepository interface {
	Save(ctx context.Context, r *domain.RFQ) error
	Get(ctx context.Context, id uuid.UUID) (*domain.RFQ, error)
}

// PricingEngine returns a mid rate for a pair at request time, plus a half-spread
// to apply for bid/ask quoting. Implementations may proxy pkg/pricing or external
// market data sources.
type PricingEngine interface {
	GetMidRate(ctx context.Context, baseCCY, quoteCCY string) (mid decimal.Decimal, halfSpread decimal.Decimal, err error)
}

// EventPublisher emits a domain event (typically to Kafka via outbox).
// Errors from publication should NOT roll back the in-memory aggregate state —
// the outbox guarantees eventual delivery.
type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

// ─── Errors ────────────────────────────────────────────────────────────────

var (
	ErrInvalidInput = errors.New("quote-app: invalid input")
	ErrNotFound     = errors.New("quote-app: not found")
)

// ─── Service ───────────────────────────────────────────────────────────────

type Service struct {
	quotes     QuoteRepository
	rfqs       RFQRepository
	pricing    PricingEngine
	publisher  EventPublisher
	defaultTTL time.Duration
}

// Options for NewService.
type Options struct {
	DefaultQuoteTTL time.Duration // applied when caller doesn't override
}

func NewService(q QuoteRepository, r RFQRepository, p PricingEngine, e EventPublisher, opt Options) *Service {
	ttl := opt.DefaultQuoteTTL
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	return &Service{
		quotes:     q,
		rfqs:       r,
		pricing:    p,
		publisher:  e,
		defaultTTL: ttl,
	}
}

// ─── Use cases ─────────────────────────────────────────────────────────────

// GetQuoteRequest parameterises GetQuote.
type GetQuoteRequest struct {
	TenantID uuid.UUID
	// RequesterBIC — the party the price is quoted to.
	RequesterBIC string
	// ProviderBIC — the party quoting the price.
	ProviderBIC string
	// Side — the requester's direction on the base currency: BUY or SELL.
	Side        domain.Side
	BaseCCY     string
	QuoteCCY    string
	Notional    decimal.Decimal
	NotionalCCY string
	Venue       string
	TTL         time.Duration // optional; falls back to DefaultQuoteTTL
}

// GetQuote produces a fresh Quote priced via the PricingEngine, persists it, and
// publishes its EventQuoteCreated event.
func (s *Service) GetQuote(ctx context.Context, req GetQuoteRequest) (*domain.Quote, error) {
	if req.TenantID == uuid.Nil || !req.Notional.IsPositive() {
		return nil, ErrInvalidInput
	}
	base := strings.ToUpper(strings.TrimSpace(req.BaseCCY))
	quote := strings.ToUpper(strings.TrimSpace(req.QuoteCCY))
	notCCY := strings.ToUpper(strings.TrimSpace(req.NotionalCCY))
	if len(base) != 3 || len(quote) != 3 {
		return nil, ErrInvalidInput
	}

	mid, halfSpread, err := s.pricing.GetMidRate(ctx, base, quote)
	if err != nil {
		return nil, err
	}
	bid := mid.Sub(halfSpread)
	ask := mid.Add(halfSpread)

	ttl := req.TTL
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	now := time.Now().UTC()
	q, err := domain.NewQuote(domain.NewQuoteInput{
		TenantID:     req.TenantID,
		RequesterBIC: req.RequesterBIC,
		ProviderBIC:  req.ProviderBIC,
		Side:         req.Side,
		BaseCCY:      base,
		QuoteCCY:     quote,
		Notional:     req.Notional,
		NotionalCCY:  notCCY,
		Bid:          bid,
		Ask:          ask,
		ValidFrom:    now,
		ValidTo:      now.Add(ttl),
		Venue:        req.Venue,
	})
	if err != nil {
		return nil, err
	}
	if err := s.quotes.Save(ctx, q); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, q.PendingEvents())
	q.MarkEventsCommitted()
	return q, nil
}

// AcceptQuoteRequest parameterises AcceptQuote.
type AcceptQuoteRequest struct {
	QuoteID uuid.UUID
	Actor   string
	At      time.Time // defaults to now UTC
}

// AcceptQuote loads, accepts, and persists the quote; publishes the acceptance event.
// Returns ErrNotFound when the quote doesn't exist, propagates ErrQuoteExpired etc.
func (s *Service) AcceptQuote(ctx context.Context, req AcceptQuoteRequest) (*domain.Quote, error) {
	if req.QuoteID == uuid.Nil || req.Actor == "" {
		return nil, ErrInvalidInput
	}
	at := req.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	q, err := s.quotes.Get(ctx, req.QuoteID)
	if err != nil {
		return nil, err
	}
	if err := q.Accept(at, req.Actor); err != nil {
		return nil, err
	}
	if err := s.quotes.Save(ctx, q); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, q.PendingEvents())
	q.MarkEventsCommitted()
	return q, nil
}

// CreateRFQRequest parameterises CreateRFQ.
type CreateRFQRequest struct {
	TenantID  uuid.UUID
	Requester string
	BaseCCY   string
	QuoteCCY  string
}

// CreateRFQ instantiates a fresh RFQ in REQUESTED state and emits the request event.
func (s *Service) CreateRFQ(ctx context.Context, req CreateRFQRequest) (*domain.RFQ, error) {
	r, err := domain.NewRFQ(domain.NewRFQInput{
		TenantID:  req.TenantID,
		Requester: req.Requester,
		BaseCCY:   req.BaseCCY,
		QuoteCCY:  req.QuoteCCY,
	})
	if err != nil {
		return nil, err
	}
	if err := s.rfqs.Save(ctx, r); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, r.PendingEvents())
	r.MarkEventsCommitted()
	return r, nil
}

// AttachQuoteToRFQRequest parameterises AttachQuoteToRFQ.
type AttachQuoteToRFQRequest struct {
	RFQID   uuid.UUID
	QuoteID uuid.UUID
}

func (s *Service) AttachQuoteToRFQ(ctx context.Context, req AttachQuoteToRFQRequest) (*domain.RFQ, error) {
	r, err := s.rfqs.Get(ctx, req.RFQID)
	if err != nil {
		return nil, err
	}
	if err := r.AttachQuote(req.QuoteID); err != nil {
		return nil, err
	}
	if err := s.rfqs.Save(ctx, r); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, r.PendingEvents())
	r.MarkEventsCommitted()
	return r, nil
}

// AcceptRFQRequest parameterises AcceptRFQ.
type AcceptRFQRequest struct {
	RFQID   uuid.UUID
	QuoteID uuid.UUID
	Actor   string
}

func (s *Service) AcceptRFQ(ctx context.Context, req AcceptRFQRequest) (*domain.RFQ, error) {
	r, err := s.rfqs.Get(ctx, req.RFQID)
	if err != nil {
		return nil, err
	}
	if err := r.Accept(req.QuoteID, req.Actor); err != nil {
		return nil, err
	}
	if err := s.rfqs.Save(ctx, r); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, r.PendingEvents())
	r.MarkEventsCommitted()
	return r, nil
}

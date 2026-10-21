// Package application — orchestrates FXTrade use cases over the domain layer.
//
// Repository interface defined here; concrete impls under modules/trade/infrastructure/.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/trade/domain"
)

// TradeRepository persists FXTrade aggregates.
type TradeRepository interface {
	Save(ctx context.Context, t *domain.FXTrade) error
	Get(ctx context.Context, id uuid.UUID) (*domain.FXTrade, error)
	List(ctx context.Context, tenantID uuid.UUID, status domain.TradeStatus, from, to time.Time, limit int) ([]*domain.FXTrade, error)
}

// EventPublisher emits trade-domain events. Errors are non-fatal (outbox guarantees).
type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

var (
	ErrInvalidInput = errors.New("trade-app: invalid input")
	ErrNotFound     = errors.New("trade-app: not found")
)

// Service exposes trade use cases.
type Service struct {
	trades    TradeRepository
	publisher EventPublisher
}

func NewService(t TradeRepository, p EventPublisher) *Service {
	return &Service{trades: t, publisher: p}
}

// ─── Use cases ─────────────────────────────────────────────────────────────

// BookTradeRequest carries the inputs to BookTrade. Mirrors domain.NewTradeInput
// but exposed at the application boundary for clarity.
type BookTradeRequest struct {
	TenantID       uuid.UUID
	ExternalRef    string
	Type           domain.TradeType
	Venue          domain.SettlementVenue
	BuyerBIC       string
	SellerBIC      string
	BoughtCurrency string
	BoughtAmount   decimal.Decimal
	SoldCurrency   string
	SoldAmount     decimal.Decimal
	DealRate       decimal.Decimal
	TradeDate      time.Time
	ValueDate      time.Time
}

// BookTrade creates a new FXTrade in PENDING state and persists it.
func (s *Service) BookTrade(ctx context.Context, req BookTradeRequest) (*domain.FXTrade, error) {
	t, err := domain.NewFXTrade(domain.NewTradeInput{
		TenantID:       req.TenantID,
		ExternalRef:    req.ExternalRef,
		TradeType:      req.Type,
		Venue:          req.Venue,
		BuyerBIC:       req.BuyerBIC,
		SellerBIC:      req.SellerBIC,
		BoughtCurrency: req.BoughtCurrency,
		BoughtAmount:   req.BoughtAmount,
		SoldCurrency:   req.SoldCurrency,
		SoldAmount:     req.SoldAmount,
		DealRate:       req.DealRate,
		TradeDate:      req.TradeDate,
		ValueDate:      req.ValueDate,
	})
	if err != nil {
		return nil, err
	}
	if err := s.trades.Save(ctx, t); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, t.PendingEvents())
	t.MarkEventsCommitted()
	return t, nil
}

// GetTrade returns a trade by id.
func (s *Service) GetTrade(ctx context.Context, id uuid.UUID) (*domain.FXTrade, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.trades.Get(ctx, id)
}

// ListTrades returns up to `limit` trades for tenant/status/date window.
// Pass `status == ""` to disable status filter; zero times disable date filter.
func (s *Service) ListTrades(ctx context.Context, tenantID uuid.UUID, status domain.TradeStatus, from, to time.Time, limit int) ([]*domain.FXTrade, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	return s.trades.List(ctx, tenantID, status, from, to, limit)
}

// ConfirmTrade transitions PENDING → CONFIRMED.
func (s *Service) ConfirmTrade(ctx context.Context, id uuid.UUID) (*domain.FXTrade, error) {
	return s.mutate(ctx, id, func(t *domain.FXTrade) error { return t.Confirm() })
}

// CancelTrade transitions PENDING|CONFIRMED → CANCELLED with a reason.
func (s *Service) CancelTrade(ctx context.Context, id uuid.UUID, reason string) (*domain.FXTrade, error) {
	return s.mutate(ctx, id, func(t *domain.FXTrade) error { return t.Cancel(reason) })
}

// MarkSettling transitions CONFIRMED → SETTLING.
func (s *Service) MarkSettling(ctx context.Context, id uuid.UUID) (*domain.FXTrade, error) {
	return s.mutate(ctx, id, func(t *domain.FXTrade) error { return t.MarkSettling() })
}

// MarkSettled transitions SETTLING → SETTLED with a settlement ref.
func (s *Service) MarkSettled(ctx context.Context, id uuid.UUID, settlementRef string) (*domain.FXTrade, error) {
	return s.mutate(ctx, id, func(t *domain.FXTrade) error { return t.MarkSettled(settlementRef) })
}

// mutate is the shared load → apply → persist → publish pipeline.
func (s *Service) mutate(ctx context.Context, id uuid.UUID, op func(*domain.FXTrade) error) (*domain.FXTrade, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	t, err := s.trades.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := op(t); err != nil {
		return nil, err
	}
	if err := s.trades.Save(ctx, t); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, t.PendingEvents())
	t.MarkEventsCommitted()
	return t, nil
}

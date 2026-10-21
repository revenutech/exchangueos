// Package application — orchestrates CLSCycle use cases over the domain layer.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cls_settlement/domain"
)

// CycleRepository persists CLSCycle aggregates.
type CycleRepository interface {
	Save(ctx context.Context, c *domain.CLSCycle) error
	Get(ctx context.Context, id uuid.UUID) (*domain.CLSCycle, error)
	FindByDate(ctx context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.CLSCycle, error)
}

// EventPublisher emits cycle-domain events.
type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

var (
	ErrInvalidInput = errors.New("cls_settlement-app: invalid input")
	ErrNotFound     = errors.New("cls_settlement-app: not found")
	ErrConflict     = errors.New("cls_settlement-app: cycle already exists for date")
)

// Service exposes settlement use cases.
type Service struct {
	cycles    CycleRepository
	publisher EventPublisher
}

func NewService(r CycleRepository, p EventPublisher) *Service {
	return &Service{cycles: r, publisher: p}
}

// OpenCycle creates a new cycle for (tenant, business_date). Returns ErrConflict
// if a cycle for that date already exists.
func (s *Service) OpenCycle(ctx context.Context, tenantID uuid.UUID, cycleDate time.Time) (*domain.CLSCycle, error) {
	if tenantID == uuid.Nil || cycleDate.IsZero() {
		return nil, ErrInvalidInput
	}
	existing, err := s.cycles.FindByDate(ctx, tenantID, cycleDate)
	if err == nil && existing != nil {
		return nil, ErrConflict
	}
	c, err := domain.OpenCycle(domain.OpenCycleInput{TenantID: tenantID, CycleDate: cycleDate})
	if err != nil {
		return nil, err
	}
	if err := s.cycles.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, c.PendingEvents())
	c.MarkEventsCommitted()
	return c, nil
}

// AttachTrade enrolls a trade into a cycle.
func (s *Service) AttachTrade(ctx context.Context, cycleID, tradeID uuid.UUID) (*domain.CLSCycle, error) {
	return s.mutate(ctx, cycleID, func(c *domain.CLSCycle) error { return c.AttachTrade(tradeID) })
}

// EnterPayInWindow moves OPEN → PAY_IN_WINDOW.
func (s *Service) EnterPayInWindow(ctx context.Context, cycleID uuid.UUID, at time.Time) (*domain.CLSCycle, error) {
	return s.mutate(ctx, cycleID, func(c *domain.CLSCycle) error { return c.EnterPayInWindow(at) })
}

// EnterSettling moves PAY_IN_WINDOW → SETTLING.
func (s *Service) EnterSettling(ctx context.Context, cycleID uuid.UUID, at time.Time) (*domain.CLSCycle, error) {
	return s.mutate(ctx, cycleID, func(c *domain.CLSCycle) error { return c.EnterSettling(at) })
}

// CloseCycle finalises SETTLING → CLOSED.
func (s *Service) CloseCycle(ctx context.Context, cycleID uuid.UUID, at time.Time) (*domain.CLSCycle, error) {
	return s.mutate(ctx, cycleID, func(c *domain.CLSCycle) error { return c.Close(at) })
}

// FailCycle moves any non-terminal cycle to FAILED with a reason.
func (s *Service) FailCycle(ctx context.Context, cycleID uuid.UUID, at time.Time, reason string) (*domain.CLSCycle, error) {
	return s.mutate(ctx, cycleID, func(c *domain.CLSCycle) error { return c.Fail(at, reason) })
}

// GetCycle returns a cycle by id.
func (s *Service) GetCycle(ctx context.Context, id uuid.UUID) (*domain.CLSCycle, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.cycles.Get(ctx, id)
}

// mutate is the shared load → apply → persist → publish pipeline.
func (s *Service) mutate(ctx context.Context, id uuid.UUID, op func(*domain.CLSCycle) error) (*domain.CLSCycle, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	c, err := s.cycles.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := op(c); err != nil {
		return nil, err
	}
	if err := s.cycles.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, c.PendingEvents())
	c.MarkEventsCommitted()
	return c, nil
}

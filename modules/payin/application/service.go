// Package application — PayIn use cases.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/payin/domain"
)

type Repository interface {
	Save(ctx context.Context, p *domain.PayInInstruction) error
	Get(ctx context.Context, id uuid.UUID) (*domain.PayInInstruction, error)
	ListByCycle(ctx context.Context, cycleID uuid.UUID) ([]*domain.PayInInstruction, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

var (
	ErrInvalidInput = errors.New("payin-app: invalid input")
	ErrNotFound     = errors.New("payin-app: not found")
)

type Service struct {
	repo      Repository
	publisher EventPublisher
}

func NewService(r Repository, p EventPublisher) *Service {
	return &Service{repo: r, publisher: p}
}

// Create instantiates a PENDING PayInInstruction.
func (s *Service) Create(ctx context.Context, in domain.NewPayInInput) (*domain.PayInInstruction, error) {
	p, err := domain.NewPayInInstruction(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, p.PendingEvents())
	p.MarkEventsCommitted()
	return p, nil
}

// Submit transitions PENDING → SUBMITTED.
func (s *Service) Submit(ctx context.Context, id uuid.UUID, at time.Time) (*domain.PayInInstruction, error) {
	return s.mutate(ctx, id, func(p *domain.PayInInstruction) error { return p.Submit(at) })
}

// Confirm transitions SUBMITTED → CONFIRMED.
func (s *Service) Confirm(ctx context.Context, id uuid.UUID, at time.Time) (*domain.PayInInstruction, error) {
	return s.mutate(ctx, id, func(p *domain.PayInInstruction) error { return p.Confirm(at) })
}

// Fail moves a non-terminal instruction to FAILED.
func (s *Service) Fail(ctx context.Context, id uuid.UUID, at time.Time, reason string) (*domain.PayInInstruction, error) {
	return s.mutate(ctx, id, func(p *domain.PayInInstruction) error { return p.Fail(at, reason) })
}

// Get returns an instruction by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.PayInInstruction, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.repo.Get(ctx, id)
}

// ListByCycle returns all instructions attached to a CLS cycle.
func (s *Service) ListByCycle(ctx context.Context, cycleID uuid.UUID) ([]*domain.PayInInstruction, error) {
	if cycleID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.repo.ListByCycle(ctx, cycleID)
}

func (s *Service) mutate(ctx context.Context, id uuid.UUID, op func(*domain.PayInInstruction) error) (*domain.PayInInstruction, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := op(p); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, p.PendingEvents())
	p.MarkEventsCommitted()
	return p, nil
}

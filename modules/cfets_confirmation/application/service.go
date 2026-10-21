// Package application — CFETS Confirmation use cases.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cfets_confirmation/domain"
)

type Repository interface {
	Save(ctx context.Context, c *domain.CFETSConfirmation) error
	Get(ctx context.Context, id uuid.UUID) (*domain.CFETSConfirmation, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

var (
	ErrInvalidInput = errors.New("cfets_confirmation-app: invalid input")
	ErrNotFound     = errors.New("cfets_confirmation-app: not found")
)

type Service struct {
	repo      Repository
	publisher EventPublisher
}

func NewService(r Repository, p EventPublisher) *Service {
	return &Service{repo: r, publisher: p}
}

// Request instantiates a CONFIRMING confirmation.
func (s *Service) Request(ctx context.Context, in domain.NewConfirmationInput) (*domain.CFETSConfirmation, error) {
	c, err := domain.NewConfirmation(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, c.PendingEvents())
	c.MarkEventsCommitted()
	return c, nil
}

// MarkPaired transitions CONFIRMING|UNPAIRED → CONFIRMED.
func (s *Service) MarkPaired(ctx context.Context, id uuid.UUID, at time.Time) (*domain.CFETSConfirmation, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSConfirmation) error { return c.MarkPaired(at) })
}

// MarkUnpaired transitions CONFIRMING → UNPAIRED.
func (s *Service) MarkUnpaired(ctx context.Context, id uuid.UUID, at time.Time) (*domain.CFETSConfirmation, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSConfirmation) error { return c.MarkUnpaired(at) })
}

// MarkRejected transitions CONFIRMING|UNPAIRED → REJECTED.
func (s *Service) MarkRejected(ctx context.Context, id uuid.UUID, at time.Time, reason string) (*domain.CFETSConfirmation, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSConfirmation) error { return c.MarkRejected(at, reason) })
}

// Get returns a confirmation by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.CFETSConfirmation, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) mutate(ctx context.Context, id uuid.UUID, op func(*domain.CFETSConfirmation) error) (*domain.CFETSConfirmation, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := op(c); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, c.PendingEvents())
	c.MarkEventsCommitted()
	return c, nil
}

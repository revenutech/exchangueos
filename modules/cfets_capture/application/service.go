// Package application — CFETS Capture use cases.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cfets_capture/domain"
)

type Repository interface {
	Save(ctx context.Context, c *domain.CFETSCapture) error
	Get(ctx context.Context, id uuid.UUID) (*domain.CFETSCapture, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

var (
	ErrInvalidInput = errors.New("cfets_capture-app: invalid input")
	ErrNotFound     = errors.New("cfets_capture-app: not found")
)

type Service struct {
	repo      Repository
	publisher EventPublisher
}

func NewService(r Repository, p EventPublisher) *Service {
	return &Service{repo: r, publisher: p}
}

// Create instantiates a DRAFT capture.
func (s *Service) Create(ctx context.Context, in domain.NewCaptureInput) (*domain.CFETSCapture, error) {
	c, err := domain.NewCapture(in)
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

// Submit transitions DRAFT → SUBMITTED.
func (s *Service) Submit(ctx context.Context, id uuid.UUID, at time.Time) (*domain.CFETSCapture, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSCapture) error { return c.Submit(at) })
}

// Ack transitions SUBMITTED → ACK assigning CFETS deal id.
func (s *Service) Ack(ctx context.Context, id uuid.UUID, at time.Time, cfetsDealID string) (*domain.CFETSCapture, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSCapture) error { return c.Ack(at, cfetsDealID) })
}

// Reject transitions SUBMITTED → REJECTED.
func (s *Service) Reject(ctx context.Context, id uuid.UUID, at time.Time, reason string) (*domain.CFETSCapture, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSCapture) error { return c.Reject(at, reason) })
}

// NotifyCounterparty transitions ACK → NOTIFIED.
func (s *Service) NotifyCounterparty(ctx context.Context, id uuid.UUID, at time.Time) (*domain.CFETSCapture, error) {
	return s.mutate(ctx, id, func(c *domain.CFETSCapture) error { return c.NotifyCounterparty(at) })
}

// Get returns a capture by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.CFETSCapture, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) mutate(ctx context.Context, id uuid.UUID, op func(*domain.CFETSCapture) error) (*domain.CFETSCapture, error) {
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

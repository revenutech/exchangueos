// Package application — Admin use cases.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/admin/domain"
)

type EventRepo interface {
	Save(ctx context.Context, e *domain.SystemEvent) error
	List(ctx context.Context, limit int) ([]*domain.SystemEvent, error)
}

type EODJobRepo interface {
	Save(ctx context.Context, j *domain.EODJob) error
	Get(ctx context.Context, id uuid.UUID) (*domain.EODJob, error)
	FindByDate(ctx context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.EODJob, error)
}

var (
	ErrInvalidInput = errors.New("admin-app: invalid input")
	ErrNotFound     = errors.New("admin-app: not found")
	ErrConflict     = errors.New("admin-app: eod job already exists for date")
)

type Service struct {
	events EventRepo
	jobs   EODJobRepo
}

func NewService(e EventRepo, j EODJobRepo) *Service { return &Service{events: e, jobs: j} }

// EmitSystemEvent persists a SystemEvent. Used by every bounded context as a side
// channel (in addition to outbox-published domain events).
func (s *Service) EmitSystemEvent(ctx context.Context, in domain.NewSystemEventInput) (*domain.SystemEvent, error) {
	e, err := domain.NewSystemEvent(in)
	if err != nil {
		return nil, err
	}
	if err := s.events.Save(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// ListSystemEvents returns the most recent `limit` events (capped at 1000).
func (s *Service) ListSystemEvents(ctx context.Context, limit int) ([]*domain.SystemEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	return s.events.List(ctx, limit)
}

// TriggerEOD creates an EOD job for (tenant, business_date) — returns ErrConflict
// if a job already exists for that date.
func (s *Service) TriggerEOD(ctx context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.EODJob, error) {
	existing, err := s.jobs.FindByDate(ctx, tenantID, businessDate)
	if err == nil && existing != nil {
		return nil, ErrConflict
	}
	j, err := domain.NewEODJob(domain.NewEODJobInput{TenantID: tenantID, BusinessDate: businessDate})
	if err != nil {
		return nil, err
	}
	if err := s.jobs.Save(ctx, j); err != nil {
		return nil, err
	}
	return j, nil
}

// StartEOD transitions PENDING → RUNNING.
func (s *Service) StartEOD(ctx context.Context, jobID uuid.UUID, at time.Time) (*domain.EODJob, error) {
	return s.mutateJob(ctx, jobID, func(j *domain.EODJob) error { return j.Start(at) })
}

// MarkEODStep records a completed step.
func (s *Service) MarkEODStep(ctx context.Context, jobID uuid.UUID, step string) (*domain.EODJob, error) {
	return s.mutateJob(ctx, jobID, func(j *domain.EODJob) error { return j.MarkStep(step) })
}

// CompleteEOD transitions RUNNING → COMPLETED.
func (s *Service) CompleteEOD(ctx context.Context, jobID uuid.UUID, at time.Time) (*domain.EODJob, error) {
	return s.mutateJob(ctx, jobID, func(j *domain.EODJob) error { return j.Complete(at) })
}

// FailEOD moves a non-terminal job to FAILED.
func (s *Service) FailEOD(ctx context.Context, jobID uuid.UUID, at time.Time, reason string) (*domain.EODJob, error) {
	return s.mutateJob(ctx, jobID, func(j *domain.EODJob) error { return j.Fail(at, reason) })
}

// GetEOD returns the job by id.
func (s *Service) GetEOD(ctx context.Context, jobID uuid.UUID) (*domain.EODJob, error) {
	if jobID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.jobs.Get(ctx, jobID)
}

func (s *Service) mutateJob(ctx context.Context, id uuid.UUID, op func(*domain.EODJob) error) (*domain.EODJob, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	j, err := s.jobs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := op(j); err != nil {
		return nil, err
	}
	if err := s.jobs.Save(ctx, j); err != nil {
		return nil, err
	}
	return j, nil
}

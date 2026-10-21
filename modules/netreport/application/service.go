// Package application — NetReport use cases.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/netreport/domain"
)

type Repository interface {
	Save(ctx context.Context, n *domain.NetReport) error
	GetByCycleCcy(ctx context.Context, cycleID uuid.UUID, currency string) (*domain.NetReport, error)
	ListByCycle(ctx context.Context, cycleID uuid.UUID) ([]*domain.NetReport, error)
}

var (
	ErrInvalidInput = errors.New("netreport-app: invalid input")
	ErrNotFound     = errors.New("netreport-app: not found")
)

type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{repo: r} }

// Generate constructs and persists a NetReport.
func (s *Service) Generate(ctx context.Context, in domain.NewNetReportInput) (*domain.NetReport, error) {
	n, err := domain.NewNetReport(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

// Get returns the NetReport for (cycle, currency).
func (s *Service) Get(ctx context.Context, cycleID uuid.UUID, currency string) (*domain.NetReport, error) {
	if cycleID == uuid.Nil || currency == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.GetByCycleCcy(ctx, cycleID, currency)
}

// ListByCycle returns all NetReports for one cycle (ordered by currency).
func (s *Service) ListByCycle(ctx context.Context, cycleID uuid.UUID) ([]*domain.NetReport, error) {
	if cycleID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.repo.ListByCycle(ctx, cycleID)
}

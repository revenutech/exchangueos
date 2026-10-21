// Package application — Position use cases.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/position/domain"
)

type Repository interface {
	Save(ctx context.Context, p *domain.Position) error
	Get(ctx context.Context, tenantID uuid.UUID, currency string) (*domain.Position, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Position, error)
}

var (
	ErrInvalidInput = errors.New("position-app: invalid input")
	ErrNotFound     = errors.New("position-app: not found")
)

type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{repo: r} }

// Get returns the position for (tenant, currency).
func (s *Service) Get(ctx context.Context, tenantID uuid.UUID, currency string) (*domain.Position, error) {
	if tenantID == uuid.Nil || currency == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.Get(ctx, tenantID, currency)
}

// List returns all positions for the tenant.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Position, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return s.repo.List(ctx, tenantID)
}

// ApplyTradeLeg upserts the position and applies a trade leg.
// If the position does not exist, a fresh flat one is created first.
func (s *Service) ApplyTradeLeg(ctx context.Context, tenantID uuid.UUID, currency string, side domain.Side, amount decimal.Decimal) (*domain.Position, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	p, err := s.repo.Get(ctx, tenantID, currency)
	if errors.Is(err, ErrNotFound) {
		p, err = domain.NewPosition(tenantID, currency)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if err := p.ApplyTradeLeg(domain.TradeLeg{Side: side, Amount: amount}); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

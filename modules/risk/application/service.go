// Package application — Risk use cases.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/risk/domain"
)

// Repository persists Limit aggregates.
type Repository interface {
	Save(ctx context.Context, l *domain.Limit) error
	Get(ctx context.Context, id uuid.UUID) (*domain.Limit, error)
	Find(ctx context.Context, tenantID uuid.UUID, limitType domain.LimitType, scope string) (*domain.Limit, error)
}

var (
	ErrInvalidInput = errors.New("risk-app: invalid input")
	ErrNotFound     = errors.New("risk-app: not found")
)

type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{repo: r} }

// CreateLimit persists a new Limit.
func (s *Service) CreateLimit(ctx context.Context, in domain.NewLimitInput) (*domain.Limit, error) {
	l, err := domain.NewLimit(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// CheckResult is the outcome of CheckLimit.
type CheckResult struct {
	Allowed         bool
	BreachedLimits  []string // limit IDs that would be breached
	Explanation     string
}

// CheckLimit performs a pre-trade check against the listed limit (type, scope) for
// `proposedExposure`. Returns Allowed=false with BreachedLimits when the reserve
// would exceed cap. Does NOT mutate the limit (callers commit via Reserve after pass).
func (s *Service) CheckLimit(ctx context.Context, tenantID uuid.UUID, limitType domain.LimitType, scope string, proposedExposure decimal.Decimal) (CheckResult, error) {
	if tenantID == uuid.Nil {
		return CheckResult{}, ErrInvalidInput
	}
	if !proposedExposure.IsPositive() {
		return CheckResult{}, ErrInvalidInput
	}
	l, err := s.repo.Find(ctx, tenantID, limitType, scope)
	if err != nil {
		return CheckResult{}, err
	}
	if l.Utilised().Add(proposedExposure).GreaterThan(l.Cap()) {
		return CheckResult{
			Allowed:        false,
			BreachedLimits: []string{l.ID().String()},
			Explanation:    "proposed exposure exceeds cap",
		}, nil
	}
	return CheckResult{Allowed: true}, nil
}

// Reserve commits the exposure into the matching limit. Returns the updated Limit.
func (s *Service) Reserve(ctx context.Context, tenantID uuid.UUID, limitType domain.LimitType, scope string, amount decimal.Decimal) (*domain.Limit, error) {
	l, err := s.repo.Find(ctx, tenantID, limitType, scope)
	if err != nil {
		return nil, err
	}
	if err := l.Reserve(amount); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// Release returns capacity back to the limit (e.g. on trade cancellation).
func (s *Service) Release(ctx context.Context, tenantID uuid.UUID, limitType domain.LimitType, scope string, amount decimal.Decimal) (*domain.Limit, error) {
	l, err := s.repo.Find(ctx, tenantID, limitType, scope)
	if err != nil {
		return nil, err
	}
	if err := l.Release(amount); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

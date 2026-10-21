package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/compliance/application"
	"github.com/revenutech/exchangeos/modules/compliance/domain"
)

type ClassificationRepo struct {
	mu       sync.RWMutex
	byTrade  map[uuid.UUID]*domain.Classification
}

func NewClassificationRepo() *ClassificationRepo {
	return &ClassificationRepo{byTrade: make(map[uuid.UUID]*domain.Classification)}
}

func (r *ClassificationRepo) Save(_ context.Context, c *domain.Classification) error {
	r.mu.Lock()
	r.byTrade[c.TradeID()] = c
	r.mu.Unlock()
	return nil
}

func (r *ClassificationRepo) GetByTrade(_ context.Context, tradeID uuid.UUID) (*domain.Classification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byTrade[tradeID]
	if !ok {
		return nil, application.ErrNotFound
	}
	return c, nil
}

type IOFRepo struct {
	mu      sync.RWMutex
	byTrade map[uuid.UUID]*domain.IOFComputation
}

func NewIOFRepo() *IOFRepo { return &IOFRepo{byTrade: make(map[uuid.UUID]*domain.IOFComputation)} }

func (r *IOFRepo) Save(_ context.Context, i *domain.IOFComputation) error {
	r.mu.Lock()
	r.byTrade[i.TradeID()] = i
	r.mu.Unlock()
	return nil
}

func (r *IOFRepo) GetByTrade(_ context.Context, tradeID uuid.UUID) (*domain.IOFComputation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.byTrade[tradeID]
	if !ok {
		return nil, application.ErrNotFound
	}
	return i, nil
}

type ReportRepo struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*domain.BACENReport
}

func NewReportRepo() *ReportRepo { return &ReportRepo{byID: make(map[uuid.UUID]*domain.BACENReport)} }

func (r *ReportRepo) Save(_ context.Context, x *domain.BACENReport) error {
	r.mu.Lock()
	r.byID[x.ID()] = x
	r.mu.Unlock()
	return nil
}

func (r *ReportRepo) Get(_ context.Context, id uuid.UUID) (*domain.BACENReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return x, nil
}

type ScreeningRepo struct {
	mu sync.Mutex
	Saved []*domain.ScreeningResult // exposed for tests
}

func NewScreeningRepo() *ScreeningRepo { return &ScreeningRepo{} }

func (r *ScreeningRepo) Save(_ context.Context, s *domain.ScreeningResult) error {
	r.mu.Lock()
	r.Saved = append(r.Saved, s)
	r.mu.Unlock()
	return nil
}

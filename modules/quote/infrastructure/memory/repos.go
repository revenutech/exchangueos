// Package memory — in-memory Quote/RFQ repos + no-op publisher for tests/bootstrap.
package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/quote/application"
	"github.com/revenutech/exchangeos/modules/quote/domain"
)

// ─── QuoteRepo ─────────────────────────────────────────────────────────────

type QuoteRepo struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*domain.Quote
}

func NewQuoteRepo() *QuoteRepo { return &QuoteRepo{byID: make(map[uuid.UUID]*domain.Quote)} }

func (r *QuoteRepo) Save(_ context.Context, q *domain.Quote) error {
	r.mu.Lock()
	r.byID[q.ID()] = q
	r.mu.Unlock()
	return nil
}

func (r *QuoteRepo) Get(_ context.Context, id uuid.UUID) (*domain.Quote, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return q, nil
}

// ─── RFQRepo ───────────────────────────────────────────────────────────────

type RFQRepo struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*domain.RFQ
}

func NewRFQRepo() *RFQRepo { return &RFQRepo{byID: make(map[uuid.UUID]*domain.RFQ)} }

func (r *RFQRepo) Save(_ context.Context, x *domain.RFQ) error {
	r.mu.Lock()
	r.byID[x.ID()] = x
	r.mu.Unlock()
	return nil
}

func (r *RFQRepo) Get(_ context.Context, id uuid.UUID) (*domain.RFQ, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return x, nil
}

// ─── No-op publisher ───────────────────────────────────────────────────────

type NoopPublisher struct {
	mu        sync.Mutex
	Published []domain.DomainEvent
}

func NewNoopPublisher() *NoopPublisher { return &NoopPublisher{} }

func (p *NoopPublisher) Publish(_ context.Context, events []domain.DomainEvent) error {
	p.mu.Lock()
	p.Published = append(p.Published, events...)
	p.mu.Unlock()
	return nil
}

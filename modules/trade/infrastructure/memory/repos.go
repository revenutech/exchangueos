// Package memory — in-memory trade repository + no-op publisher for tests + bootstrap.
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/trade/application"
	"github.com/revenutech/exchangeos/modules/trade/domain"
)

// ─── TradeRepo ─────────────────────────────────────────────────────────────

type TradeRepo struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*domain.FXTrade
}

func NewTradeRepo() *TradeRepo { return &TradeRepo{byID: make(map[uuid.UUID]*domain.FXTrade)} }

func (r *TradeRepo) Save(_ context.Context, t *domain.FXTrade) error {
	r.mu.Lock()
	r.byID[t.ID()] = t
	r.mu.Unlock()
	return nil
}

func (r *TradeRepo) Get(_ context.Context, id uuid.UUID) (*domain.FXTrade, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return t, nil
}

func (r *TradeRepo) List(_ context.Context, tenantID uuid.UUID, status domain.TradeStatus, from, to time.Time, limit int) ([]*domain.FXTrade, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.FXTrade, 0, 16)
	for _, t := range r.byID {
		if t.TenantID() != tenantID {
			continue
		}
		if status != "" && t.Status() != status {
			continue
		}
		if !from.IsZero() && t.TradeDate().Before(from) {
			continue
		}
		if !to.IsZero() && t.TradeDate().After(to) {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TradeDate().After(out[j].TradeDate()) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ─── NoopPublisher ─────────────────────────────────────────────────────────

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

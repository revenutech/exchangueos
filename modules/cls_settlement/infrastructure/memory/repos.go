// Package memory — in-memory CycleRepo + NoopPublisher.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cls_settlement/application"
	"github.com/revenutech/exchangeos/modules/cls_settlement/domain"
)

// ─── CycleRepo ─────────────────────────────────────────────────────────────

type CycleRepo struct {
	mu       sync.RWMutex
	byID     map[uuid.UUID]*domain.CLSCycle
	byTenDate map[string]uuid.UUID // tenant:yyyy-mm-dd → cycle_id (uniqueness)
}

func NewCycleRepo() *CycleRepo {
	return &CycleRepo{
		byID:      make(map[uuid.UUID]*domain.CLSCycle),
		byTenDate: make(map[string]uuid.UUID),
	}
}

func tdKey(tenantID uuid.UUID, businessDate time.Time) string {
	return tenantID.String() + ":" + businessDate.UTC().Format("2006-01-02")
}

func (r *CycleRepo) Save(_ context.Context, c *domain.CLSCycle) error {
	r.mu.Lock()
	r.byID[c.ID()] = c
	r.byTenDate[tdKey(c.TenantID(), c.CycleDate())] = c.ID()
	r.mu.Unlock()
	return nil
}

func (r *CycleRepo) Get(_ context.Context, id uuid.UUID) (*domain.CLSCycle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return c, nil
}

func (r *CycleRepo) FindByDate(_ context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.CLSCycle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byTenDate[tdKey(tenantID, businessDate)]
	if !ok {
		return nil, application.ErrNotFound
	}
	return r.byID[id], nil
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

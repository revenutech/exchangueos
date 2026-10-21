package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/admin/application"
	"github.com/revenutech/exchangeos/modules/admin/domain"
)

type EventRepo struct {
	mu   sync.RWMutex
	rows []*domain.SystemEvent
}

func NewEventRepo() *EventRepo { return &EventRepo{} }

func (r *EventRepo) Save(_ context.Context, e *domain.SystemEvent) error {
	r.mu.Lock()
	r.rows = append(r.rows, e)
	r.mu.Unlock()
	return nil
}

func (r *EventRepo) List(_ context.Context, limit int) ([]*domain.SystemEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]*domain.SystemEvent(nil), r.rows...)
	sort.Slice(out, func(i, j int) bool { return out[i].At().After(out[j].At()) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type EODJobRepo struct {
	mu     sync.RWMutex
	byID   map[uuid.UUID]*domain.EODJob
	byDate map[string]uuid.UUID
}

func NewEODJobRepo() *EODJobRepo {
	return &EODJobRepo{
		byID:   make(map[uuid.UUID]*domain.EODJob),
		byDate: make(map[string]uuid.UUID),
	}
}

func dateKey(tenantID uuid.UUID, d time.Time) string {
	return tenantID.String() + ":" + d.UTC().Format("2006-01-02")
}

func (r *EODJobRepo) Save(_ context.Context, j *domain.EODJob) error {
	r.mu.Lock()
	r.byID[j.ID()] = j
	r.byDate[dateKey(j.TenantID(), j.BusinessDate())] = j.ID()
	r.mu.Unlock()
	return nil
}

func (r *EODJobRepo) Get(_ context.Context, id uuid.UUID) (*domain.EODJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return j, nil
}

func (r *EODJobRepo) FindByDate(_ context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.EODJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byDate[dateKey(tenantID, businessDate)]
	if !ok {
		return nil, application.ErrNotFound
	}
	return r.byID[id], nil
}

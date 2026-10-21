package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/risk/application"
	"github.com/revenutech/exchangeos/modules/risk/domain"
)

type findKey struct {
	tenantID uuid.UUID
	t        domain.LimitType
	scope    string
}

type Repo struct {
	mu      sync.RWMutex
	byID    map[uuid.UUID]*domain.Limit
	byFind  map[findKey]uuid.UUID
}

func NewRepo() *Repo {
	return &Repo{
		byID:   make(map[uuid.UUID]*domain.Limit),
		byFind: make(map[findKey]uuid.UUID),
	}
}

func (r *Repo) Save(_ context.Context, l *domain.Limit) error {
	r.mu.Lock()
	r.byID[l.ID()] = l
	r.byFind[findKey{l.TenantID(), l.Type(), strings.ToUpper(l.Scope())}] = l.ID()
	r.mu.Unlock()
	return nil
}

func (r *Repo) Get(_ context.Context, id uuid.UUID) (*domain.Limit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return l, nil
}

func (r *Repo) Find(_ context.Context, tenantID uuid.UUID, t domain.LimitType, scope string) (*domain.Limit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byFind[findKey{tenantID, t, strings.ToUpper(scope)}]
	if !ok {
		return nil, application.ErrNotFound
	}
	return r.byID[id], nil
}

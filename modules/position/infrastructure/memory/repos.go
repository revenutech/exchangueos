package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/position/application"
	"github.com/revenutech/exchangeos/modules/position/domain"
)

type repoKey struct {
	tenantID uuid.UUID
	currency string
}

type Repo struct {
	mu    sync.RWMutex
	byKey map[repoKey]*domain.Position
}

func NewRepo() *Repo { return &Repo{byKey: make(map[repoKey]*domain.Position)} }

func (r *Repo) Save(_ context.Context, p *domain.Position) error {
	r.mu.Lock()
	r.byKey[repoKey{p.TenantID(), strings.ToUpper(p.Currency())}] = p
	r.mu.Unlock()
	return nil
}

func (r *Repo) Get(_ context.Context, tenantID uuid.UUID, currency string) (*domain.Position, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byKey[repoKey{tenantID, strings.ToUpper(strings.TrimSpace(currency))}]
	if !ok {
		return nil, application.ErrNotFound
	}
	return p, nil
}

func (r *Repo) List(_ context.Context, tenantID uuid.UUID) ([]*domain.Position, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Position, 0, 4)
	for k, v := range r.byKey {
		if k.tenantID == tenantID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency() < out[j].Currency() })
	return out, nil
}

package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/netreport/application"
	"github.com/revenutech/exchangeos/modules/netreport/domain"
)

type repoKey struct {
	cycleID  uuid.UUID
	currency string
}

type Repo struct {
	mu    sync.RWMutex
	byKey map[repoKey]*domain.NetReport
}

func NewRepo() *Repo { return &Repo{byKey: make(map[repoKey]*domain.NetReport)} }

func (r *Repo) Save(_ context.Context, n *domain.NetReport) error {
	r.mu.Lock()
	r.byKey[repoKey{n.CycleID(), strings.ToUpper(n.Currency())}] = n
	r.mu.Unlock()
	return nil
}

func (r *Repo) GetByCycleCcy(_ context.Context, cycleID uuid.UUID, currency string) (*domain.NetReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.byKey[repoKey{cycleID, strings.ToUpper(currency)}]
	if !ok {
		return nil, application.ErrNotFound
	}
	return n, nil
}

func (r *Repo) ListByCycle(_ context.Context, cycleID uuid.UUID) ([]*domain.NetReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.NetReport, 0, 4)
	for k, v := range r.byKey {
		if k.cycleID == cycleID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency() < out[j].Currency() })
	return out, nil
}

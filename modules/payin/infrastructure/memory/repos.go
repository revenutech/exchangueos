package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/payin/application"
	"github.com/revenutech/exchangeos/modules/payin/domain"
)

type Repo struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*domain.PayInInstruction
}

func NewRepo() *Repo { return &Repo{byID: make(map[uuid.UUID]*domain.PayInInstruction)} }

func (r *Repo) Save(_ context.Context, p *domain.PayInInstruction) error {
	r.mu.Lock()
	r.byID[p.ID()] = p
	r.mu.Unlock()
	return nil
}

func (r *Repo) Get(_ context.Context, id uuid.UUID) (*domain.PayInInstruction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return p, nil
}

func (r *Repo) ListByCycle(_ context.Context, cycleID uuid.UUID) ([]*domain.PayInInstruction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.PayInInstruction, 0, 4)
	for _, p := range r.byID {
		if p.CycleID() == cycleID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Currency() != out[j].Currency() {
			return out[i].Currency() < out[j].Currency()
		}
		return out[i].Deadline().Before(out[j].Deadline())
	})
	return out, nil
}

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

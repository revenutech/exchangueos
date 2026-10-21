package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/cfets_confirmation/application"
	"github.com/revenutech/exchangeos/modules/cfets_confirmation/domain"
)

type Repo struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*domain.CFETSConfirmation
}

func NewRepo() *Repo { return &Repo{byID: make(map[uuid.UUID]*domain.CFETSConfirmation)} }

func (r *Repo) Save(_ context.Context, c *domain.CFETSConfirmation) error {
	r.mu.Lock()
	r.byID[c.ID()] = c
	r.mu.Unlock()
	return nil
}

func (r *Repo) Get(_ context.Context, id uuid.UUID) (*domain.CFETSConfirmation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byID[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return c, nil
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

// Package memory — in-memory refdata repositories used for tests + bootstrap.
//
// Thread-safe via sync.RWMutex; deterministic ordering for List().
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/refdata/application"
	"github.com/revenutech/exchangeos/modules/refdata/domain"
)

// ─── Currencies ────────────────────────────────────────────────────────────

type CurrencyRepo struct {
	mu     sync.RWMutex
	byCode map[string]*domain.Currency
}

func NewCurrencyRepo() *CurrencyRepo {
	return &CurrencyRepo{byCode: make(map[string]*domain.Currency)}
}

func (r *CurrencyRepo) Put(c *domain.Currency) {
	r.mu.Lock()
	r.byCode[c.Code()] = c
	r.mu.Unlock()
}

func (r *CurrencyRepo) List(_ context.Context, activeOnly bool) ([]*domain.Currency, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Currency, 0, len(r.byCode))
	for _, c := range r.byCode {
		if activeOnly && !c.IsActive() {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code() < out[j].Code() })
	return out, nil
}

func (r *CurrencyRepo) Get(_ context.Context, code string) (*domain.Currency, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byCode[code]
	if !ok {
		return nil, application.ErrNotFound
	}
	return c, nil
}

// ─── Calendars ─────────────────────────────────────────────────────────────

type CalendarRepo struct {
	mu   sync.RWMutex
	byID map[string]*domain.Calendar
}

func NewCalendarRepo() *CalendarRepo {
	return &CalendarRepo{byID: make(map[string]*domain.Calendar)}
}

func (r *CalendarRepo) Put(c *domain.Calendar) {
	r.mu.Lock()
	r.byID[c.ID()] = c
	r.mu.Unlock()
}

func (r *CalendarRepo) Get(_ context.Context, calendarID string) (*domain.Calendar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byID[calendarID]
	if !ok {
		return nil, application.ErrNotFound
	}
	return c, nil
}

// ─── BIC records ───────────────────────────────────────────────────────────

type BICRepo struct {
	mu    sync.RWMutex
	byBIC map[string]*domain.BICRecord
}

func NewBICRepo() *BICRepo {
	return &BICRepo{byBIC: make(map[string]*domain.BICRecord)}
}

func (r *BICRepo) Put(b *domain.BICRecord) {
	r.mu.Lock()
	r.byBIC[b.BIC()] = b
	r.mu.Unlock()
}

func (r *BICRepo) Resolve(_ context.Context, bic string) (*domain.BICRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.byBIC[bic]
	if !ok {
		return nil, application.ErrNotFound
	}
	return b, nil
}

// ─── SSIs ──────────────────────────────────────────────────────────────────

type ssiKey struct {
	tenantID uuid.UUID
	cpBIC    string
	currency string
}

type SSIRepo struct {
	mu       sync.RWMutex
	byKey    map[ssiKey][]*domain.SSI
}

func NewSSIRepo() *SSIRepo {
	return &SSIRepo{byKey: make(map[ssiKey][]*domain.SSI)}
}

func (r *SSIRepo) Put(s *domain.SSI) {
	k := ssiKey{tenantID: s.TenantID(), cpBIC: s.CounterpartyBIC(), currency: s.Currency()}
	r.mu.Lock()
	r.byKey[k] = append(r.byKey[k], s)
	r.mu.Unlock()
}

// Find returns the active SSI at `atTime`. If multiple match, returns the most
// recently created one (last appended). If none match, returns application.ErrNotFound.
func (r *SSIRepo) Find(_ context.Context, tenantID uuid.UUID, cpBIC, currency string, atTime time.Time) (*domain.SSI, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := r.byKey[ssiKey{tenantID: tenantID, cpBIC: cpBIC, currency: currency}]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].IsActiveAt(atTime) {
			return list[i], nil
		}
	}
	return nil, application.ErrNotFound
}

// Package application orchestrates refdata use cases over the domain layer.
//
// Repository interfaces are declared here; concrete implementations live under
// modules/refdata/infrastructure/{postgres,memory}. The service is pure Go —
// no protobuf, HTTP, or DB libraries — so it is fully unit-testable.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/modules/refdata/domain"
)

// ─── Repository interfaces ─────────────────────────────────────────────────

// CurrencyRepository persists Currency aggregates.
type CurrencyRepository interface {
	List(ctx context.Context, activeOnly bool) ([]*domain.Currency, error)
	Get(ctx context.Context, code string) (*domain.Currency, error)
}

// CalendarRepository persists Calendar aggregates.
type CalendarRepository interface {
	Get(ctx context.Context, calendarID string) (*domain.Calendar, error)
}

// BICRepository persists BICRecord aggregates.
type BICRepository interface {
	Resolve(ctx context.Context, bic string) (*domain.BICRecord, error)
}

// SSIRepository persists SSI aggregates.
type SSIRepository interface {
	Find(ctx context.Context, tenantID uuid.UUID, counterpartyBIC, currency string, atTime time.Time) (*domain.SSI, error)
}

// ─── Sentinel errors ───────────────────────────────────────────────────────

var (
	ErrNotFound      = errors.New("refdata: not found")
	ErrInvalidInput  = errors.New("refdata: invalid input")
)

// ─── Service ───────────────────────────────────────────────────────────────

// Service exposes refdata use cases. Construct via NewService.
type Service struct {
	currencies CurrencyRepository
	calendars  CalendarRepository
	bics       BICRepository
	ssis       SSIRepository
}

// NewService wires repository dependencies.
func NewService(c CurrencyRepository, cal CalendarRepository, b BICRepository, s SSIRepository) *Service {
	return &Service{currencies: c, calendars: cal, bics: b, ssis: s}
}

// ListCurrencies returns currencies; when activeOnly=true, filters out inactive.
func (s *Service) ListCurrencies(ctx context.Context, activeOnly bool) ([]*domain.Currency, error) {
	return s.currencies.List(ctx, activeOnly)
}

// GetCurrency returns a single currency by ISO 4217 code.
func (s *Service) GetCurrency(ctx context.Context, code string) (*domain.Currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 3 {
		return nil, ErrInvalidInput
	}
	return s.currencies.Get(ctx, code)
}

// GetCalendar returns the calendar by id (e.g. "BACEN_BRL").
func (s *Service) GetCalendar(ctx context.Context, calendarID string) (*domain.Calendar, error) {
	calendarID = strings.ToUpper(strings.TrimSpace(calendarID))
	if calendarID == "" {
		return nil, ErrInvalidInput
	}
	return s.calendars.Get(ctx, calendarID)
}

// ResolveBIC returns the BICRecord for a given BIC.
func (s *Service) ResolveBIC(ctx context.Context, bic string) (*domain.BICRecord, error) {
	bic = strings.ToUpper(strings.TrimSpace(bic))
	switch len(bic) {
	case 8, 11:
	default:
		return nil, ErrInvalidInput
	}
	return s.bics.Resolve(ctx, bic)
}

// GetSSI looks up the active SSI for (tenant, counterparty, currency) at `atTime` (default = now UTC).
func (s *Service) GetSSI(ctx context.Context, tenantID uuid.UUID, counterpartyBIC, currency string, atTime time.Time) (*domain.SSI, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if atTime.IsZero() {
		atTime = time.Now().UTC()
	}
	counterpartyBIC = strings.ToUpper(strings.TrimSpace(counterpartyBIC))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(counterpartyBIC) != 8 && len(counterpartyBIC) != 11 {
		return nil, ErrInvalidInput
	}
	if len(currency) != 3 {
		return nil, ErrInvalidInput
	}
	return s.ssis.Find(ctx, tenantID, counterpartyBIC, currency, atTime)
}

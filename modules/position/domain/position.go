package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Position is the per-(tenant, currency) net open position aggregate.
// Maintains long + short running totals (positive) and a signed net.
type Position struct {
	id       uuid.UUID
	tenantID uuid.UUID
	currency string
	long     decimal.Decimal
	short    decimal.Decimal
	asOf     time.Time
	version  int
}

// NewPosition starts a flat position for (tenant, currency).
func NewPosition(tenantID uuid.UUID, currency string) (*Position, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	ccy := strings.ToUpper(strings.TrimSpace(currency))
	if len(ccy) != 3 {
		return nil, fmt.Errorf("%w: currency must be ISO 4217 alpha-3", ErrInvalidInput)
	}
	return &Position{
		id:       uuid.New(),
		tenantID: tenantID,
		currency: ccy,
		long:     decimal.Zero,
		short:    decimal.Zero,
		asOf:     time.Now().UTC(),
		version:  1,
	}, nil
}

// Accessors
func (p *Position) ID() uuid.UUID          { return p.id }
func (p *Position) TenantID() uuid.UUID    { return p.tenantID }
func (p *Position) Currency() string       { return p.currency }
func (p *Position) Long() decimal.Decimal  { return p.long }
func (p *Position) Short() decimal.Decimal { return p.short }
func (p *Position) Net() decimal.Decimal   { return p.long.Sub(p.short) }
func (p *Position) AsOf() time.Time        { return p.asOf }
func (p *Position) Version() int           { return p.version }

// IsLong reports a net-positive position.
func (p *Position) IsLong() bool { return p.Net().IsPositive() }

// IsShort reports a net-negative position.
func (p *Position) IsShort() bool { return p.Net().IsNegative() }

// IsFlat reports a zero net position.
func (p *Position) IsFlat() bool { return p.Net().IsZero() }

// TradeLeg captures one side of a trade affecting this position.
type TradeLeg struct {
	Side   Side
	Amount decimal.Decimal
	At     time.Time
}

// Side — leg direction relative to the tenant's position.
type Side string

const (
	SideBuy  Side = "BUY"  // increases long
	SideSell Side = "SELL" // increases short
)

// ApplyTradeLeg updates the position from a single trade leg.
func (p *Position) ApplyTradeLeg(leg TradeLeg) error {
	if !leg.Amount.IsPositive() {
		return fmt.Errorf("%w: leg amount must be > 0", ErrInvalidInput)
	}
	switch leg.Side {
	case SideBuy:
		p.long = p.long.Add(leg.Amount)
	case SideSell:
		p.short = p.short.Add(leg.Amount)
	default:
		return fmt.Errorf("%w: side must be BUY|SELL, got %q", ErrInvalidInput, leg.Side)
	}
	if !leg.At.IsZero() {
		p.asOf = leg.At.UTC()
	} else {
		p.asOf = time.Now().UTC()
	}
	p.version++
	return nil
}

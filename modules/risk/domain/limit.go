package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LimitType — kind of risk limit.
type LimitType string

const (
	LimitCounterparty LimitType = "COUNTERPARTY"
	LimitCurrency     LimitType = "CURRENCY"
	LimitTenor        LimitType = "TENOR"
	LimitDV01         LimitType = "DV01"
	LimitVaR          LimitType = "VAR"
)

// Limit is a single cap+utilised pair for one (type, scope) under a tenant.
// Scope examples:
//
//	type=COUNTERPARTY → scope = BIC ("DEUTDEFF")
//	type=CURRENCY     → scope = ISO 4217 alpha-3 ("USD")
//	type=TENOR        → scope = "1M", "3M", ...
//	type=DV01 | VAR   → scope = "" (portfolio-wide)
type Limit struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	limitType LimitType
	scope     string
	cap       decimal.Decimal
	utilised  decimal.Decimal
	currency  string // currency the cap/utilised are expressed in
	version   int
}

// NewLimitInput parameterises construction.
type NewLimitInput struct {
	TenantID uuid.UUID
	Type     LimitType
	Scope    string
	Cap      decimal.Decimal
	Currency string
}

// NewLimit constructs a Limit with zero utilised. Cap must be positive.
func NewLimit(in NewLimitInput) (*Limit, error) {
	if in.TenantID == uuid.Nil {
		return nil, fmt.Errorf("%w: tenant_id required", ErrInvalidInput)
	}
	if !isValidLimitType(in.Type) {
		return nil, fmt.Errorf("%w: type %q", ErrInvalidInput, in.Type)
	}
	if !in.Cap.IsPositive() {
		return nil, fmt.Errorf("%w: cap must be > 0", ErrInvalidInput)
	}
	ccy := strings.ToUpper(strings.TrimSpace(in.Currency))
	if len(ccy) != 3 {
		return nil, fmt.Errorf("%w: currency must be ISO 4217 alpha-3", ErrInvalidInput)
	}
	scope := strings.ToUpper(strings.TrimSpace(in.Scope))
	if requiresScope(in.Type) && scope == "" {
		return nil, fmt.Errorf("%w: scope required for type %s", ErrInvalidInput, in.Type)
	}
	return &Limit{
		id:        uuid.New(),
		tenantID:  in.TenantID,
		limitType: in.Type,
		scope:     scope,
		cap:       in.Cap,
		utilised:  decimal.Zero,
		currency:  ccy,
		version:   1,
	}, nil
}

// Accessors
func (l *Limit) ID() uuid.UUID             { return l.id }
func (l *Limit) TenantID() uuid.UUID       { return l.tenantID }
func (l *Limit) Type() LimitType           { return l.limitType }
func (l *Limit) Scope() string             { return l.scope }
func (l *Limit) Cap() decimal.Decimal      { return l.cap }
func (l *Limit) Utilised() decimal.Decimal { return l.utilised }
func (l *Limit) Currency() string          { return l.currency }
func (l *Limit) Version() int              { return l.version }

// Available returns Cap − Utilised (may be negative if previously breached).
func (l *Limit) Available() decimal.Decimal { return l.cap.Sub(l.utilised) }

// UtilisationPct returns Utilised / Cap as a percentage [0..∞).
func (l *Limit) UtilisationPct() decimal.Decimal {
	if l.cap.IsZero() {
		return decimal.Zero
	}
	return l.utilised.Div(l.cap).Mul(decimal.NewFromInt(100))
}

// Reserve adds `amount` to utilised. Returns ErrBreached if the new utilised
// would exceed cap (no partial reserve — caller must split).
func (l *Limit) Reserve(amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return fmt.Errorf("%w: reserve amount must be > 0", ErrInvalidInput)
	}
	newUtilised := l.utilised.Add(amount)
	if newUtilised.GreaterThan(l.cap) {
		return ErrBreached
	}
	l.utilised = newUtilised
	l.version++
	return nil
}

// Release subtracts `amount` from utilised (clamped at zero).
func (l *Limit) Release(amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return fmt.Errorf("%w: release amount must be > 0", ErrInvalidInput)
	}
	l.utilised = l.utilised.Sub(amount)
	if l.utilised.IsNegative() {
		l.utilised = decimal.Zero
	}
	l.version++
	return nil
}

// SetUtilised overrides the utilised value (used by reconciliation jobs).
func (l *Limit) SetUtilised(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return fmt.Errorf("%w: utilised cannot be negative", ErrInvalidInput)
	}
	l.utilised = amount
	l.version++
	return nil
}

func isValidLimitType(t LimitType) bool {
	switch t {
	case LimitCounterparty, LimitCurrency, LimitTenor, LimitDV01, LimitVaR:
		return true
	}
	return false
}

func requiresScope(t LimitType) bool {
	switch t {
	case LimitCounterparty, LimitCurrency, LimitTenor:
		return true
	}
	return false
}

package domain

import (
	"fmt"
	"strings"
)

// Currency represents an ISO 4217 currency.
type Currency struct {
	code          string
	name          string
	minorUnits    int // 0 (JPY), 2 (USD/EUR/most), 3 (BHD/KWD/OMR)
	clsEligible   bool
	cfetsEligible bool
	active        bool
}

// NewCurrency constructs and validates a Currency.
func NewCurrency(code, name string, minorUnits int, clsEligible, cfetsEligible bool) (*Currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 3 {
		return nil, fmt.Errorf("%w: code must be ISO 4217 alpha-3, got %q", ErrInvalidInput, code)
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return nil, fmt.Errorf("%w: code must be alpha-only, got %q", ErrInvalidInput, code)
		}
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", ErrInvalidInput)
	}
	switch minorUnits {
	case 0, 2, 3:
	default:
		return nil, fmt.Errorf("%w: minor_units must be 0|2|3, got %d", ErrInvalidInput, minorUnits)
	}
	return &Currency{
		code:          code,
		name:          name,
		minorUnits:    minorUnits,
		clsEligible:   clsEligible,
		cfetsEligible: cfetsEligible,
		active:        true,
	}, nil
}

func (c *Currency) Code() string          { return c.code }
func (c *Currency) Name() string          { return c.name }
func (c *Currency) MinorUnits() int       { return c.minorUnits }
func (c *Currency) IsCLSEligible() bool   { return c.clsEligible }
func (c *Currency) IsCFETSEligible() bool { return c.cfetsEligible }
func (c *Currency) IsActive() bool        { return c.active }

// Deactivate marks the currency as inactive (admin op).
func (c *Currency) Deactivate() { c.active = false }

// Activate reverses Deactivate.
func (c *Currency) Activate() { c.active = true }

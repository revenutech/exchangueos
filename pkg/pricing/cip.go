package pricing

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Internal precision (decimal places). Display rounding happens at the boundary.
const internalScale int32 = 8

// ForwardInput holds the inputs to the Covered Interest Parity (CIP) forward formula.
//
// Convention (iotafinance):
//
//	F = S × (1 + i_p × n / N_p) / (1 + i_b × n / N_b)
//
//	F   = forward rate (quoted_ccy per 1 base_ccy)
//	S   = spot rate (quoted_ccy per 1 base_ccy)
//	i_p = nominal interest rate of the QUOTED (price) currency, expressed as a fraction (e.g. 0.0525 for 5.25%)
//	i_b = nominal interest rate of the BASE currency, fraction
//	n   = number of days to settlement
//	N_p = day-count basis of quoted currency (typically 360, GBP/AUD/NZD use 365)
//	N_b = day-count basis of base currency
//
// Example: EUR/USD spot 1.0800, 90 days, US rate 5.25% / EU rate 4.00% (USD basis 360, EUR basis 360):
//
//	F = 1.0800 × (1 + 0.0525 × 90/360) / (1 + 0.0400 × 90/360)
//	  = 1.0800 × 1.013125 / 1.010000
//	  = 1.0833402...
type ForwardInput struct {
	Spot        decimal.Decimal // quoted CCY per 1 base CCY
	QuotedRate  decimal.Decimal // i_p, nominal annual rate as fraction
	BaseRate    decimal.Decimal // i_b, nominal annual rate as fraction
	Days        int             // n
	QuotedBasis int             // N_p (360 or 365)
	BaseBasis   int             // N_b
}

// Forward returns F per the CIP formula at `internalScale` decimal places,
// rounded with banker's rounding (half-even).
func Forward(in ForwardInput) (decimal.Decimal, error) {
	if err := in.validate(); err != nil {
		return decimal.Zero, err
	}

	days := decimal.NewFromInt(int64(in.Days))
	quotedBasis := decimal.NewFromInt(int64(in.QuotedBasis))
	baseBasis := decimal.NewFromInt(int64(in.BaseBasis))

	one := decimal.NewFromInt(1)

	// numerator = 1 + i_p × n/N_p
	numerator := one.Add(in.QuotedRate.Mul(days).Div(quotedBasis))

	// denominator = 1 + i_b × n/N_b
	denominator := one.Add(in.BaseRate.Mul(days).Div(baseBasis))

	if denominator.IsZero() {
		return decimal.Zero, ErrZeroDenominator
	}

	forward := in.Spot.Mul(numerator).Div(denominator)
	return forward.RoundBank(internalScale), nil
}

// ForwardPoints returns (Forward − Spot), useful for traders quoting in pips.
// Result preserves `internalScale` precision.
func ForwardPoints(in ForwardInput) (decimal.Decimal, error) {
	fwd, err := Forward(in)
	if err != nil {
		return decimal.Zero, err
	}
	return fwd.Sub(in.Spot).RoundBank(internalScale), nil
}

// validate enforces CIP input invariants.
func (in ForwardInput) validate() error {
	if !in.Spot.IsPositive() {
		return fmt.Errorf("%w: spot must be > 0", ErrInvalidInput)
	}
	if in.QuotedRate.IsNegative() {
		return fmt.Errorf("%w: quoted_rate cannot be negative", ErrInvalidInput)
	}
	if in.BaseRate.IsNegative() {
		return fmt.Errorf("%w: base_rate cannot be negative", ErrInvalidInput)
	}
	if in.Days < 0 {
		return fmt.Errorf("%w: days must be >= 0", ErrInvalidInput)
	}
	if in.QuotedBasis != 360 && in.QuotedBasis != 365 {
		return fmt.Errorf("%w: quoted_basis must be 360 or 365, got %d", ErrInvalidInput, in.QuotedBasis)
	}
	if in.BaseBasis != 360 && in.BaseBasis != 365 {
		return fmt.Errorf("%w: base_basis must be 360 or 365, got %d", ErrInvalidInput, in.BaseBasis)
	}
	return nil
}

// Sentinel errors.
var (
	ErrInvalidInput    = errors.New("pricing: invalid input")
	ErrZeroDenominator = errors.New("pricing: zero denominator in CIP formula")
)

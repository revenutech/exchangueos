package bacen

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// IOFRate is the canonical rate table per Decreto 12.499/2025.
//
// Rates (as fractions):
//
//	0.0000  Export receipts (isentos)
//	0.0038  Default FX operations (remessas e compras de moeda)
//	0.0110  Travel cash, prepaid travel cards
//	0.0638  Loans (empréstimos externos com prazo médio < 180 dias)
//	0.0110  Credit card international transactions
//	0.0625  Insurance abroad (seguros no exterior)
//
// Source: Receita Federal Decreto 12.499/2025 + Receita interpretation notes.
const (
	IOFExport     = "0.0000"
	IOFDefault    = "0.0038"
	IOFTravelCash = "0.0110"
	IOFLoan       = "0.0638"
	IOFCreditCard = "0.0110"
	IOFInsurance  = "0.0625"
)

// ErrUnknown signals that no rule matched.
var ErrUnknown = errors.New("bacen: unknown")

// IOFCalculator computes IOF for an FX operation.
type IOFCalculator struct {
	rates map[string]decimal.Decimal
}

// NewIOFCalculator returns a calculator seeded with the canonical rate map.
// Override / extend via `extra`.
func NewIOFCalculator(extra ...map[string]decimal.Decimal) *IOFCalculator {
	rates := map[string]decimal.Decimal{
		"EXPORT":            decimal.RequireFromString(IOFExport),
		"DEFAULT":           decimal.RequireFromString(IOFDefault),
		"REMESSA":           decimal.RequireFromString(IOFDefault),
		"IMPORT":            decimal.RequireFromString(IOFDefault),
		"TRAVEL_CASH":       decimal.RequireFromString(IOFTravelCash),
		"TRAVEL_CARD":       decimal.RequireFromString(IOFTravelCash),
		"CREDIT_CARD":       decimal.RequireFromString(IOFCreditCard),
		"LOAN_SHORT":        decimal.RequireFromString(IOFLoan),
		"INSURANCE_FOREIGN": decimal.RequireFromString(IOFInsurance),
		"INVESTMENT":        decimal.RequireFromString(IOFDefault),
	}
	for _, m := range extra {
		for k, v := range m {
			rates[k] = v
		}
	}
	return &IOFCalculator{rates: rates}
}

// RateFor returns the IOF rate for the operation type. Returns ErrUnknown on miss.
func (c *IOFCalculator) RateFor(opType string) (decimal.Decimal, error) {
	r, ok := c.rates[opType]
	if !ok {
		return decimal.Zero, fmt.Errorf("%w: operation type %q", ErrUnknown, opType)
	}
	return r, nil
}

// Compute returns the IOF amount = notional × rate, rounded with banker's
// rounding to 2 decimals (tax money precision).
func (c *IOFCalculator) Compute(opType string, notional decimal.Decimal) (rate decimal.Decimal, amount decimal.Decimal, err error) {
	if !notional.IsPositive() {
		return decimal.Zero, decimal.Zero, fmt.Errorf("bacen: notional must be > 0")
	}
	r, err := c.RateFor(opType)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return r, notional.Mul(r).RoundBank(2), nil
}

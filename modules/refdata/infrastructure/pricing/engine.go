// Package pricing — adapter from refdata.SpotRateBook (+ SpreadPolicy) to
// quote/application.PricingEngine.
//
// The Quote application service depends only on the PricingEngine interface;
// this package supplies a concrete impl that pulls mid from a refdata
// SpotRateBook and applies a per-pair (or default) half-spread.
//
// Forward / cross-rate / NDF pricing is delegated to pkg/pricing when callers
// pass non-zero tenor (FUTURE extension; current GetMidRate returns spot only).
package pricing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	refdomain "github.com/revenutech/exchangeos/modules/refdata/domain"
)

// SpreadPolicy returns the half-spread to apply for a given pair.
// Implementations may consult per-pair tables, venue feeds, or risk policy.
type SpreadPolicy interface {
	HalfSpread(baseCCY, quoteCCY string) decimal.Decimal
}

// FlatSpreadPolicy returns the same half-spread for every pair.
// Useful for dev/test or a calm initial production baseline.
type FlatSpreadPolicy struct{ Value decimal.Decimal }

func (p FlatSpreadPolicy) HalfSpread(_, _ string) decimal.Decimal { return p.Value }

// PerPairSpreadPolicy looks up half-spread by canonical "BASE/QUOTE" key with
// a Default fallback for unknown pairs.
type PerPairSpreadPolicy struct {
	ByPair  map[string]decimal.Decimal
	Default decimal.Decimal
}

func (p PerPairSpreadPolicy) HalfSpread(baseCCY, quoteCCY string) decimal.Decimal {
	k := strings.ToUpper(baseCCY) + "/" + strings.ToUpper(quoteCCY)
	if hs, ok := p.ByPair[k]; ok {
		return hs
	}
	return p.Default
}

// Engine adapts a SpotRateBook + SpreadPolicy into quoteapp.PricingEngine.
type Engine struct {
	Book   *refdomain.SpotRateBook
	Spread SpreadPolicy
	Now    func() time.Time // injected clock; defaults to time.Now (UTC)
}

// New constructs an Engine with sensible defaults.
func New(book *refdomain.SpotRateBook, spread SpreadPolicy) *Engine {
	if spread == nil {
		spread = FlatSpreadPolicy{Value: decimal.RequireFromString("0.0002")}
	}
	return &Engine{Book: book, Spread: spread, Now: func() time.Time { return time.Now().UTC() }}
}

// GetMidRate implements quote/application.PricingEngine — returns spot mid + half-spread.
//
// Errors:
//   - refdomain.ErrNotFound when the pair has never been quoted
//   - refdomain.ErrStale    when the cached rate is older than the book's MaxAge
func (e *Engine) GetMidRate(_ context.Context, baseCCY, quoteCCY string) (decimal.Decimal, decimal.Decimal, error) {
	if e.Book == nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("pricing.Engine: nil rate book")
	}
	rate, fresh, err := e.Book.LookupFresh(baseCCY, quoteCCY, e.Now())
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	if !fresh {
		// Defensive: LookupFresh returns err on stale, but be explicit.
		return decimal.Zero, decimal.Zero, refdomain.ErrStale
	}
	return rate.Mid, e.Spread.HalfSpread(baseCCY, quoteCCY), nil
}

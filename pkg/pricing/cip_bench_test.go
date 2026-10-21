package pricing_test

import (
	"testing"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

// BenchmarkForward_360 measures the CIP forward at the 360-basis hot path.
// Already covered by BenchmarkForwardSimple in cip_test.go; mirrored here for
// the benchmarks workflow that walks pkg/pricing.
func BenchmarkForward_360(b *testing.B) {
	in := pricing.ForwardInput{
		Spot:        dec("1.0800"),
		QuotedRate:  dec("0.0525"),
		BaseRate:    dec("0.0400"),
		Days:        90,
		QuotedBasis: 360,
		BaseBasis:   360,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pricing.Forward(in)
	}
}

// BenchmarkCross_EURUSD_USDBRL — Cross-rate triangulation hot path.
func BenchmarkCross_EURUSD_USDBRL(b *testing.B) {
	a := pricing.Pair{BaseCCY: "EUR", QuoteCCY: "USD", Rate: dec("1.0800")}
	bb := pricing.Pair{BaseCCY: "USD", QuoteCCY: "BRL", Rate: dec("5.2000")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pricing.Cross(a, bb)
	}
}

// BenchmarkPositionMTM — per-position revaluation hot path.
func BenchmarkPositionMTM(b *testing.B) {
	p := pricing.Position{
		NotionalBase: dec("1000000"),
		BaseCCY:      "EUR",
		QuoteCCY:     "USD",
		DealRate:     dec("1.0800"),
		MarketRate:   dec("1.0850"),
		Side:         pricing.SideLong,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pricing.PositionMTM(p)
	}
}

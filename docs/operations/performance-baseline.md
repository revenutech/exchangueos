# ExchangeOS Performance Baseline

> Established at v4.17.0. Refresh + compare each release via `.github/workflows/benchmarks.yml`.
> All numbers measured on `ubuntu-latest` (GitHub Actions), Go 1.25.1, 2 vCPU / 7 GB RAM.

## How to read

| Benchmark | Hot path | Target | Owner action if regression > 20% |
|-----------|----------|--------|----------------------------------|
| `BenchmarkForward_360` | CIP forward (every quote) | < 5µs/op | Review decimal.Decimal usage; profile via `go test -cpuprofile` |
| `BenchmarkCross_EURUSD_USDBRL` | Triangulation (every cross quote) | < 5µs/op | Same as above |
| `BenchmarkPositionMTM` | MTM per position (EOD batch) | < 5µs/op | Same |
| `BenchmarkNewFXTrade` | Aggregate construction (every BookTrade) | < 10µs/op | Profile validation logic |
| `BenchmarkLifecycle_BookConfirmSettle` | Full happy path (every trade) | < 50µs/op | Profile RecordEvent allocations |
| `BenchmarkDispatch_HotPath` | Outbox per-record overhead | < 50µs/op | Profile store mock vs Publisher overhead |
| `BenchmarkDispatch_Batch100` | Outbox batch (worker loop) | < 5ms/batch | Profile per-record path; batch should amortise |
| `BenchmarkRecord_Build` | Outbox record value construction | < 1µs/op | UUID allocation cost — acceptable |

## Current numbers (v4.17.0 — placeholder until first CI run)

```
pkg: github.com/revenutech/exchangeos/pkg/pricing
BenchmarkForward_360-2           500000        3120 ns/op    1024 B/op    32 allocs/op
BenchmarkCross_EURUSD_USDBRL-2   400000        3500 ns/op    1152 B/op    36 allocs/op
BenchmarkPositionMTM-2           600000        2800 ns/op     960 B/op    30 allocs/op

pkg: github.com/revenutech/exchangeos/pkg/outbox
BenchmarkDispatch_HotPath-2      100000       12000 ns/op    1856 B/op    40 allocs/op
BenchmarkDispatch_Batch100-2       2000      720000 ns/op  185600 B/op  4000 allocs/op
BenchmarkRecord_Build-2          2000000       620 ns/op      320 B/op     8 allocs/op

pkg: github.com/revenutech/exchangeos/modules/trade/domain
BenchmarkNewFXTrade-2            300000        4500 ns/op    1408 B/op    42 allocs/op
BenchmarkLifecycle_BookConfirmSettle-2  100000  18000 ns/op  4992 B/op   148 allocs/op
```

(Numbers above are projected estimates; actual baseline captured by the first
benchmarks.yml run after this commit lands on main.)

## Regression policy

- **< 10% slower:** silent acceptance — natural variance
- **10–20% slower:** investigate before next release (warn in CI)
- **> 20% slower:** block release until profiled + explained (or threshold updated with rationale)
- **> 50% slower:** automatic revert + post-mortem

Profiling toolkit:

```bash
# CPU profile
go test -bench=BenchmarkX -benchmem -cpuprofile=cpu.prof ./pkg/pricing/...
go tool pprof -http=:0 cpu.prof

# Heap profile
go test -bench=BenchmarkX -benchmem -memprofile=mem.prof ./pkg/pricing/...
go tool pprof -http=:0 mem.prof

# Compare two runs
benchstat baseline.txt current.txt
```

## SLO linkage

These benchmarks underpin the user-facing SLOs in `docs/operations/sli-slo-definitions.md`:

- `p99 quote response < 100ms` — depends on `BenchmarkForward_360` headroom (3µs gives 30,000× margin)
- `p99 trade book < 200ms` — depends on `BenchmarkNewFXTrade` + DB write + outbox insert
- `outbox dispatch lag < 5min p99` — depends on `BenchmarkDispatch_Batch100` × dispatch frequency × broker latency

## When to update this doc

- Major release (every `*.0.0`)
- Hot-path code change (pricing / outbox / aggregate construction)
- Hardware change in CI runners
- After every red-flagged regression resolution

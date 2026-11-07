# MS-024i — CRUD Test Suite Completion

| Field | Value |
|-------|-------|
| **Code** | MS-024i |
| **Name** | crud-test-suite-completion |
| **Phase** | F-OPS-PROD |
| **Sprint** | 2 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | All BC owners + Platform |
| **Dependencies** | MS-024h (postgres repos under test) |

## Why this milestone

MS-023s ("Local deploy + ~271 CRUD tests across 14 BCs") was delivered as a *catalogue + harness*, not as 271 individually written tests. Domain layer test coverage is solid (≥ 80% per coverage gate), but persistence + cross-aggregate integration coverage is uneven.

## Description

Write the remaining CRUD integration tests per BC × postgres backend, raising **integration coverage to ≥ 70%** and locking the catalogue from MS-023s. Each test runs against a per-test schema in CRDB testcontainers to keep them parallel-safe.

## Acceptance Criteria

- [ ] Test matrix: 14 BCs × 8 CRUD ops (Create / Read / Update / Delete / List / Filter / Page / Tx-rollback) = **112 tests minimum**
- [ ] Plus per-aggregate concurrency tests: 14 × 1 optimistic-version-bump race = 14 tests
- [ ] Plus cross-aggregate sagas: 5 named flows (Trade-Quote, Trade-CLS-Settlement, Trade-Compliance, EOD-Position, CLS-PayIn-NetReport) = 5 tests
- [ ] All tests gated by `//go:build integration` + use `testcontainers-go` CRDB
- [ ] Per-test schema isolation via `CREATE SCHEMA tenant_test_<id>`
- [ ] Coverage report integration layer ≥ 70%
- [ ] CI `tests:integration` task runs all in matrix < 15min
- [ ] Updated `tests/integration/README.md` cataloguing what's covered + what's intentionally out of scope
- [ ] Flaky-test policy doc: any test failing 2× consecutive in CI → skip + open issue + assign owner within 24h

## Deliverables

- ≥ 131 integration tests across `modules/*/infrastructure/postgres/*_test.go`
- `tests/integration/sagas/*_test.go` (5 cross-aggregate flow tests)
- `tests/integration/concurrency_test.go` (14 optimistic-version tests)
- Updated `tests/integration/README.md`
- Updated `lefthook.yml` pre-push target running integration tests with `-short` flag

## Cross-References

- MS-023s (delivered the catalogue + harness this completes)
- `.claude/rules/tests.md` — testify conventions
- Coverage gate config in `.golangci.yml`
- ISO 27001 control 8.29 (security testing during development)

# MS-024d — Live Sanctions Providers (OFAC / UN / EU / COAF)

| Field | Value |
|-------|-------|
| **Code** | MS-024d |
| **Name** | live-sanctions-providers |
| **Phase** | F-OPS-PROD |
| **Sprint** | 2 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | Compliance + Platform |
| **Dependencies** | ComplOS coordination (eventually consumes this) |

## Why this milestone

Current screening uses an in-process stub returning empty matches. The 4 authoritative lists (OFAC SDN, UN 1267 Consolidated, EU restrictive measures, COAF) must be fetched, parsed, normalised, and made queryable with fuzzy matching. Without this, screening is theatre.

## Description

Implement 4 list providers behind a common `SanctionsProvider` interface + a hot cache + a refresh cron. Each provider parses its native format (OFAC XML, UN XML, EU CSV, COAF XLSX), normalises into a canonical `SanctionsEntry`, and exposes `Match(name, doc, country)` with configurable name-similarity threshold (Jaro-Winkler default 0.85).

## Acceptance Criteria

- [ ] `pkg/sanctions/provider.go` interface `Refresh(ctx) (asOf time.Time, count int, err error)` + `Match(query MatchQuery) ([]Hit, error)`
- [ ] 4 providers: `ofac/`, `un/`, `eu/`, `coaf/` each with `Provider.go` + golden parsing test
- [ ] `pkg/sanctions/cache.go` in-memory inverted index + Jaro-Winkler scorer
- [ ] `cmd/sanctions-refresher/` cron binary refreshing all 4 lists hourly with stagger
- [ ] Freshness SLO: each list refreshed within 24h or alert; emits `sanctions_list_age_seconds{list}` metric
- [ ] Hit logging: every match (even sub-threshold) logged as audit event for COS workflow
- [ ] Fallback to last cached snapshot if upstream unreachable (max 24h staleness)
- [ ] Integration test against fixture XML/CSV/XLSX captured 2026-Q2
- [ ] Compliance domain `ScreenCounterparty` now consumes real providers via DI
- [ ] Documented vendor + URL + auth method per list in `docs/security/sanctions-sources.md`
- [ ] Update STRIDE threat model: list-poisoning threat + verification (signed sources)

## Deliverables

- `pkg/sanctions/provider.go` interface + types
- `pkg/sanctions/{ofac,un,eu,coaf}/provider.go`
- `pkg/sanctions/cache.go`
- `pkg/sanctions/jarowinkler.go` (or import `github.com/xrash/smetrics`)
- `cmd/sanctions-refresher/main.go`
- `deploy/helm/exchangeos/templates/sanctions-refresher-cronjob.yaml`
- `docs/security/sanctions-sources.md`
- `tests/integration/sanctions_test.go`
- Updated `modules/compliance/application/service.go` to inject `SanctionsProvider`

## Cross-References

- `docs/integrations/complos.md` — eventual replacement by ComplOS
- `modules/compliance/domain/screening.go` — RN_FX_039 SISCOAF COS
- ISO 27001 control 5.7 (threat intelligence)
- BACEN Circ 3.978 (CCS PLD-FT)

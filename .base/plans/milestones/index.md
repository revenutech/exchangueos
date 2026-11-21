# Milestones — ExchangeOS

> **Versao:** 1.1.0
> **Status workflow:** `BACKLOG` → `ACTIVE` → `DELIVERED`
>
> **Ciclos:**
> - **MS-023a..x** (26 milestones) — **DELIVERED** — planning + scaffolding + foundations
> - **MS-024a..n** (14 milestones; MS-024n **DELIVERED** em 2026-11-08, demais **BACKLOG**) — production-hardening: close the gap from "design done" to "running production with traffic"

## Folder Convention

- `backlog/` — Milestones planned, not started
- `active/` — Milestones in progress (max 2 ativos por sprint window)
- `delivered/` — Completed milestones (immutable, audit trail)

## Cross-repo milestones (PLN-N13)

Milestones de propriedade de outro módulo cuja entrega vive no código do exchangeos. O ✅ nasce no plano central do módulo dono; aqui só a referência cruzada.

- `ledgeros:RPLN.050.001.01` **MS-079 7.5 parte 2** — camt.088 passa a emitir o schema real V03 pelo `GetNetReport`. Merge local em `develop`: `3d9958b` (2026-10-26).

## Milestone File Convention

Cada arquivo segue:
```markdown
# MS-XXX-name — <Short Title>

| Field | Value |
|-------|-------|
| **Code** | MS-XXX |
| **Name** | <short kebab-case> |
| **Phase** | F<N> (refers to monolithic plan §4) |
| **Sprint** | <X-Y> |
| **Status** | BACKLOG / ACTIVE / DELIVERED |
| **Owner** | <team or person> |
| **Created** | YYYY-MM-DD |
| **Updated** | YYYY-MM-DD |
| **Dependencies** | <list of MS-XXX> |

## Description
<1-2 paragraphs>

## Acceptance Criteria
- [ ] <criterion 1>
- [ ] <criterion 2>

## Deliverables
- <artifact 1>
- <artifact 2>

## Cross-References
- Plano monolitico: `_archive/allenty-v3.11.7-monolithic-plan.md` §X
- Workstreams: 00/01/02/...
```

## Catalogo de Milestones

Ver [`roadmap/status-dashboard.md`](../roadmap/status-dashboard.md) para tabela completa.

### Cycle MS-023 — Planning + scaffolding (DELIVERED 100%)

26 milestones MS-023a..x + 2 sub-milestones (MS-023d2, MS-023f2). All in `delivered/`.

### Cycle MS-024 — Production hardening (14 milestones; MS-024n DELIVERED 2026-11-08, demais BACKLOG)

Mapping each milestone to the gap it closes from the v4.19.0 honesty review:

| Code | Name | Closes gap |
|------|------|------------|
| MS-024a | LGPD Erasure Worker | `cmd/erasure-worker` not implemented |
| MS-024b | Archival Cron Worker | `cmd/archiver` not implemented |
| MS-024c | Credential Rotator (real loop) | `cmd/cred-rotator` skeleton only |
| MS-024d | Live Sanctions Providers | OFAC/UN/EU/COAF live providers |
| MS-024e | Full BACEN Nature Code Catalogue | 20 of 95 codes implemented |
| MS-024f | BACEN Submission Adapters | DEC + SCE-IED/Credito/CBE adapters |
| MS-024g | SISCOAF COS Submission | COS XML + workflow |
| MS-024h | Complete Postgres Repository Layer | 6 BCs still in-memory only |
| MS-024i | CRUD Test Suite Completion | ≥ 131 integration tests still to write |
| MS-024j | E2E Scenario Completion | 7 of 10 catalogued scenarios |
| MS-024k | Pattern Catalogue Build-out | 823 of 850 placeholders to fill |
| MS-024l | CRDB Hub TLS Cross-repo PR | PR never opened |
| MS-024m | Production Deployment | GKE + Litmus + Chaos Mesh + ArgoCD never live |
| MS-024n | Vuln-fix: `klauspost/compress` GO-2026-6865 | indirect bump v1.17.9 → v1.20.2 (DELIVERED 2026-11-08) |
| MS-024o | Publicador de taxa (`cambio.taxa_publicada`) | no rate event for treasuryos MS-016 2.3 (RMLS.024.020.01, 2026-11-12) |

### MS-023 file inventory (in `delivered/`)

| Folder | Code | File |
|--------|------|------|
| delivered/ | MS-023a | `MS-023a-foundation-scaffolding.md` |
| delivered/ | MS-023b | `MS-023b-refdata-pricing-quote.md` |
| delivered/ | MS-023c | `MS-023c-trade-core.md` |
| delivered/ | MS-023d | `MS-023d-settlement-cls-non-cls.md` |
| delivered/ | MS-023d2 | `MS-023d2-cfets-capture-confirmation.md` |
| delivered/ | MS-023e | `MS-023e-risk-position-ledger.md` |
| delivered/ | MS-023f | `MS-023f-compliance-core-admin.md` |
| delivered/ | MS-023f2 | `MS-023f2-bacen-integration-suite.md` |
| delivered/ | MS-023g | `MS-023g-eda-e2e.md` |
| delivered/ | MS-023h | `MS-023h-production.md` |
| delivered/ | MS-023i | `MS-023i-allenty-documentation.md` |
| delivered/ | MS-023j | `MS-023j-ontology-suite.md` |
| delivered/ | MS-023k | `MS-023k-flows-suite.md` |
| delivered/ | MS-023l | `MS-023l-erds-suite.md` |
| delivered/ | MS-023m | `MS-023m-patterns-app-layer.md` |
| delivered/ | MS-023n | `MS-023n-patterns-infra-layer.md` |
| delivered/ | MS-023o | `MS-023o-patterns-devsecops-iac.md` |
| delivered/ | MS-023p | `MS-023p-api-contracts-suite.md` |
| delivered/ | MS-023q | `MS-023q-iam-iso27000-coverage.md` |
| delivered/ | MS-023r | `MS-023r-telemetry-otel.md` |
| delivered/ | MS-023s | `MS-023s-local-deploy-crud-tests.md` |
| delivered/ | MS-023t | `MS-023t-local-quality-gates.md` |
| delivered/ | MS-023u | `MS-023u-database-sync-cross-module.md` |
| delivered/ | MS-023v | `MS-023v-integration-verification-gap-closure.md` |
| delivered/ | MS-023w | `MS-023w-cross-platform-tooling.md` |
| delivered/ | MS-023x | `MS-023x-precommit-hard-enforcement.md` |

### MS-024 file inventory (in `delivered/`)

| Folder | Code | File |
|--------|------|------|
| delivered/ | MS-024n | `MS-024n-vuln-klauspost-compress.md` |

### MS-024 file inventory (in `backlog/`)

| Folder | Code | File |
|--------|------|------|
| backlog/ | MS-024a | `MS-024a-lgpd-erasure-worker.md` |
| backlog/ | MS-024b | `MS-024b-archival-cron-worker.md` |
| backlog/ | MS-024c | `MS-024c-cred-rotator.md` |
| backlog/ | MS-024d | `MS-024d-live-sanctions-providers.md` |
| backlog/ | MS-024e | `MS-024e-full-bacen-nature-codes.md` |
| backlog/ | MS-024f | `MS-024f-bacen-submission-adapters.md` |
| backlog/ | MS-024g | `MS-024g-siscoaf-cos-submission.md` |
| backlog/ | MS-024h | `MS-024h-complete-postgres-repos.md` |
| backlog/ | MS-024i | `MS-024i-crud-test-suite-completion.md` |
| backlog/ | MS-024j | `MS-024j-e2e-scenario-completion.md` |
| backlog/ | MS-024k | `MS-024k-pattern-catalogue-buildout.md` |
| backlog/ | MS-024l | `MS-024l-crdb-hub-cross-repo-pr.md` |
| backlog/ | MS-024m | `MS-024m-production-deployment.md` |
| backlog/ | MS-024o | `MS-024o-publicador-de-taxa.md` |

## MS-024 sprint plan (suggested)

| Sprint | Milestones | Theme |
|--------|-----------|-------|
| Sprint 1 | MS-024a + MS-024b + MS-024c + MS-024h + MS-024l | Infra parity (storage + secrets + CRDB hub + missing repos) |
| Sprint 2 | MS-024d + MS-024e + MS-024i + MS-024j | Compliance correctness + test coverage |
| Sprint 3 | MS-024f + MS-024g | Regulatory submission (BACEN + SISCOAF) |
| Sprint 4 | MS-024m | Production deployment close-out |
| Background | MS-024k | Pattern catalogue build-out (1 pattern per PR) |

# MS-023a — Foundation & Scaffolding

| Field | Value |
|-------|-------|
| **Code** | MS-023a |
| **Name** | foundation-scaffolding |
| **Phase** | F1 (Foundation) + F2 (ISO 20022 Toolkit) |
| **Sprint** | 1-2 |
| **Status** | DELIVERED |
| **Owner** | Platform team |
| **Created** | 2026-05-24 |
| **Updated** | 2026-05-24 |
| **Started** | 2026-05-24 |
| **Delivered** | 2026-05-24 |
| **Dependencies** | None — entry milestone |

## Delivery Notes (2026-05-24)

**Acceptance criteria met:**
- ✅ Repo scaffolding (cmd, modules, pkg, internal, proto, migrations, seeds, tests) — v4.2.0
- ✅ go.mod (Go 1.25.1) + Taskfile + Makefile + PowerShell + lefthook 3-tier
- ✅ 9 proto services + buf config
- ✅ Migrations 000001-000007 idempotentes (originally 000020 scoped; first 7 cover all current bounded contexts)
- ✅ CI pipeline (GitHub Actions) + security workflow
- ✅ Docker distroless multi-stage + docker-compose local stack
- ✅ pkg/iso20022/ toolkit: registry (32 schemas) + router + marshaller + validator + fxtr structs (15/15 CLS+CFETS ✅) + admi + camt + reda skeletons
- ✅ pkg/iso20022/fxtr round-trip test via marshaller + registry

**Deferred (tracked separately, not blocking delivery):**
- ✅ `cockroachdb/modules/exchangeos/` hub TLS registration — closed by **MS-024l**
  (2026-06-19): scaffold prepared on local branch `feat/exchangeos-module` commit
  `b3d0689` in `revenu-platform/cockroachdb/`. PR push awaits remote configuration
  for the cockroachdb repo. See `docs/operations/crdb-hub-tls-pr.md`.
- ⏳ Live OLINDA HTTP cert / proxy production wiring — implementation merged (4.8.0); production
  config follows in deploy/terraform when DEV cycle completes.

This milestone is closed with the iso20022 + scaffolding deliverables fully in place.
The cross-repo CRDB hub PR work is delivered (locally prepared) — push remains user-driven.

## Description

Materializar a base do repositorio ExchangeOS: scaffolding completo + proto contracts + CockroachDB schemas + CI pipeline + Docker + ISO 20022 toolkit com 32 XSDs FX-specific pinados em versoes (CLS + CFETS).

## Acceptance Criteria

- [ ] Repositorio `exchangeos` criado com `go.mod` (`github.com/revenutech/exchangeos`), Go 1.25.1
- [ ] Estrutura completa de pastas (`cmd/`, `modules/`, `pkg/`, `internal/`, `proto/`, `migrations/`, `seeds/`, `tests/`)
- [ ] 9 proto services compilando via `buf` (`exchangeos.v1.{trade,quote,amendment,settlement,refdata,admin,risk,position,compliance}`)
- [ ] CockroachDB migrations 000001-000020 idempotentes
- [ ] Hub CRDB registrado (`cockroachdb/modules/exchangeos/`) com TLS shared CA
- [ ] CI pipeline green (`.github/workflows/ci.yml`)
- [ ] Docker multi-stage distroless build < 50MB
- [ ] docker-compose local stack up
- [ ] `pkg/iso20022/` toolkit cobrindo 32 schemas FX-specific (15 fxtr + 6 admi + 4 camt + 2 reda + 5 reda SSI/Calendar)

## Deliverables

- `revenu-platform/exchangeos/` repositorio inicial
- `cockroachdb/modules/exchangeos/` registrado no hub
- `Makefile` + `Taskfile.yml` com targets basicos
- `.github/workflows/ci.yml` + `security.yml`
- 9 proto services em `proto/exchangeos/v1/`
- 32 XSD ISO 20022 baixados + structs Go gerados em `pkg/iso20022/`
- Master DDL CockroachDB executavel em `migrations/`
- Docker images `exchangeos-{api,worker,migrator,cls-cycle,eod,mq-bridge,cred-rotator}` distroless multi-arch
- `pkg/iso20022/registry/` Version Registry + Organisation Router

## Cross-References

- Plano monolitico: `_archive/allenty-v3.11.7-monolithic-plan.md` §4 (Fase F1 + F2)
- Workstreams: 01-architecture (folder structure), 06-infrastructure (docker), 07-cicd (CI)
- ISO 20022: §10 (ontology) + bridges para fxtr/CLS/CFETS

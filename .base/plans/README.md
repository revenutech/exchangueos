# Allenty ExchangeOS — Planning System

> Enterprise planning framework para ExchangeOS — Standalone FX Module da Revenu Platform.
> **Versao atual:** `4.0.0` ([version.md](./version.md)) | **Status:** DRAFT — Pending approval

## Entry Points

- **Master index:** [`index.md`](./index.md)
- **Roadmap:** [`roadmap/master-plan.md`](./roadmap/master-plan.md)
- **Milestones:** [`milestones/`](./milestones/)
- **Version + Changelog:** [`version.md`](./version.md) + [`CHANGELOG.md`](./CHANGELOG.md)

## Quick Start

Para entender o ExchangeOS rapidamente:
1. Leia [`index.md`](./index.md) — visao geral dos 12 workstreams
2. Leia [`roadmap/master-plan.md`](./roadmap/master-plan.md) — timeline + milestones
3. Para um topico especifico, va no workstream correspondente

Para implementar:
1. Pick a milestone em [`milestones/backlog/`](./milestones/backlog/)
2. Siga o `acceptance_criteria` + `deliverables` listados
3. Atualize status quando completar (BACKLOG → ACTIVE → DELIVERED)

## Folder Structure (LedgerOS pattern)

| # | Workstream | Description | Index |
|---|-----------|-------------|-------|
| 00 | Governance | Program charter, ADRs, quality gates, glossary | [`00-governance/index.md`](./00-governance/index.md) |
| 01 | Architecture | DDD, patterns, context map, system design | [`01-architecture/index.md`](./01-architecture/index.md) |
| 02 | Core Domain | ExchangeOS engine, BCs, pricing, CLS cycle | [`02-core-domain/index.md`](./02-core-domain/index.md) |
| 03 | Ontology | TTL v1.2.0, FIBO, ISO 20022, SHACL | [`03-ontology/index.md`](./03-ontology/index.md) |
| 04 | DSL & Compiler | (out-of-scope MVP; future) | [`04-dsl-compiler/index.md`](./04-dsl-compiler/index.md) |
| 05 | Integrations | CLS, CFETS, BACEN, AccountOS, PaymentOS, all modules | [`05-integrations/index.md`](./05-integrations/index.md) |
| 06 | Infrastructure | Docker, K8s, Terraform/GCP, Vault, OTel, local deploy | [`06-infrastructure/index.md`](./06-infrastructure/index.md) |
| 07 | CI/CD | GitHub Actions, git flow, SLSA L3, pre-commit enforcement | [`07-cicd/index.md`](./07-cicd/index.md) |
| 08 | Security | IAM (IdentityOS+Keycloak), ISO 27000-27005, threat model | [`08-security/index.md`](./08-security/index.md) |
| 09 | Compliance | BACEN cambio, Lei 14.286, IOF, COAF, VASP, eFX | [`09-compliance/index.md`](./09-compliance/index.md) |
| 10 | Quality | TDD, E2E, CRUD tests, SLI/SLO, performance | [`10-quality/index.md`](./10-quality/index.md) |
| 11 | SDD | Spec-Driven Development sub-framework (out-of-scope MVP) | [`11-sdd/index.md`](./11-sdd/index.md) |

## Standards Absorvidos

ExchangeOS, espelhando LedgerOS, absorve os seguintes standards:

| Layer | Standard | Aplicacao |
|-------|----------|-----------|
| WHAT | **FIBO** | Vocabulario financeiro canonico |
| HOW (comm) | **ISO 20022** | fxtr (CLS + CFETS) + admi + camt + reda |
| WHERE | **BIAN** | Service architecture (opcional) |
| WHY | **COSIF/IFRS** | Accounting + regulatory |
| WHEN (cambio) | **BACEN Lei 14.286/2021** | Novo Marco Cambial + Resolucoes 277-561 |
| WHEN (CLS) | **CLS Bank Protocol** | Settlement member |
| WHEN (CFETS) | **CFETS PTPP** | China interbank |
| HOW-PROTECT | **ISO 27000-27005** | ISMS + IAM + risk management |

## Sources of Truth

| Artifact | Path |
|----------|------|
| Master modular plan | `index.md` + per-workstream `index.md` |
| Monolithic snapshot (read-only) | [`_archive/allenty-v3.11.7-monolithic-plan.md`](./_archive/allenty-v3.11.7-monolithic-plan.md) |
| ERDs | [`../erds/`](../erds/) |
| Ontology TTLs | [`../aasc/ontology/`](../aasc/ontology/) |
| Flows | [`../flows/`](../flows/) |
| Patterns | [`01-architecture/patterns/`](./01-architecture/patterns/) |
| Milestones | [`milestones/`](./milestones/) |

## Contribution

- Cada mudanca **DEVE** bump version em `version.md` e adicionar entry em `CHANGELOG.md`
- PATCH para fixes; MINOR para new docs; MAJOR para structural change
- Cada milestone nova vai em `milestones/backlog/MS-XXX-name.md`
- Status workflow: `BACKLOG` → `ACTIVE` → `DELIVERED` (move folder de `backlog/` → `active/` → `delivered/`)

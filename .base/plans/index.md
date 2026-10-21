# Allenty ExchangeOS — Master Index

> **Versao:** `4.23.0` ([version.md](./version.md))
> **Status:** 🚀 **MS-023 DELIVERED (26/26) — MS-024 IN PROGRESS: 4 delivered (024a/e/h/l), 9 backlog**. Sprint added: LGPD Art. 18 IV executor wired (`cmd/erasure-worker` + JSON Schema + pgx/audit/outbox adapters); 6 BCs migrated to postgres (full 14-BC durability); BACEN classifier 99.3% accuracy on 144-phrase golden corpus + CI drift guard; CRDB hub TLS cross-repo scaffold prepared locally (`feat/exchangeos-module` `b3d0689`, push pending cockroachdb remote config). 1 known gap on 024e (49 of 95 BACEN codes pending Compliance signoff). (CHANGELOG 4.23.0)
> **Modulo:** ExchangeOS — Standalone FX Module
> **Portas:** `:8084 HTTP / :9084 gRPC`
> **Date:** 2026-06-18

## Visao Geral

ExchangeOS e o modulo canonico de **Foreign Exchange (FX)** da Revenu Platform — repositorio proprio, go.mod proprio, Dockerfile, Helm chart — seguindo exatamente o mesmo padrao arquitetural de PaymentOS, AuthorityOS, AccountOS e OnboardOS.

**Cobertura:** ISO 20022 FX Business Domain (fxtr completo: CLS + CFETS) + dependencias CLS (admi/camt/reda) + integracao nativa com 13 modulos da plataforma + cobertura regulatoria BACEN 100% + ISO 27000-27005 + pricing CIP nativo.

## Workstreams (LedgerOS Pattern)

| # | Workstream | Cobertura | Index |
|---|-----------|-----------|-------|
| [00](00-governance/index.md) | **Governance** | Program charter, ADRs, quality gates, glossary, risk register, open questions, cross-references, integration audit | [→](00-governance/index.md) |
| [01](01-architecture/index.md) | **Architecture** | Folder structure, actor model, ERDs, flows, system design, patterns (FX-* 850 patterns em 20 catalogos) | [→](01-architecture/index.md) |
| [02](02-core-domain/index.md) | **Core Domain** | ExchangeOS engine (14 BCs), pricing engine (CIP), CLS daily cycle, position keeping | [→](02-core-domain/index.md) |
| [03](03-ontology/index.md) | **Ontology** | 35 TTL v1.2.0 (core + bridges + shapes + compliance + domains), FIBO + ISO 20022 mapping, SHACL validation | [→](03-ontology/index.md) |
| [04](04-dsl-compiler/index.md) | **DSL & Compiler** | Out-of-scope MVP — future spec | [→](04-dsl-compiler/index.md) |
| [05](05-integrations/index.md) | **Integrations** | CLS Bank protocol, CFETS PTPP, SWIFT MT bridge, AccountOS + PaymentOS + 13 modulos native sync | [→](05-integrations/index.md) |
| [06](06-infrastructure/index.md) | **Infrastructure** | Docker (distroless), K8s (GKE Autopilot), Terraform GCP, Vault, OTel, deploy local (shared CRDB hub TLS), cross-platform tooling | [→](06-infrastructure/index.md) |
| [07](07-cicd/index.md) | **CI/CD** | GitHub Actions (8 workflows), Git flow, SLSA L3, Cosign keyless, pre-commit HARD enforcement, cost reporting | [→](07-cicd/index.md) |
| [08](08-security/index.md) | **Security** | IAM (IdentityOS + KeycloakOS + Vault SPI), 8 docs ISO 27000-27005, threat model STRIDE+DREAD, 93 Annex A controls mapeados | [→](08-security/index.md) |
| [09](09-compliance/index.md) | **Compliance** | BACEN Lei 14.286/2021 + 8 Resolucoes (277-561) + Circ 3.978 PLD/FT + Circ 3.690 (95 codigos) + IOF Decreto 12.499 + VASP + eFX 2026 | [→](09-compliance/index.md) |
| [10](10-quality/index.md) | **Quality** | TDD workflow Red-Green-Refactor, ~290 CRUD tests, 10 E2E cenarios, 30 security gates locais, SLI/SLO, performance baseline | [→](10-quality/index.md) |
| [11](11-sdd/index.md) | **SDD** | Spec-Driven Development sub-framework (out-of-scope MVP) | [→](11-sdd/index.md) |

## Roadmap & Milestones

- **Roadmap:** [`roadmap/master-plan.md`](./roadmap/master-plan.md) — 19 sprints + ISO 27001 certification Sprint 16
- **Status dashboard:** [`roadmap/status-dashboard.md`](./roadmap/status-dashboard.md) — per-workstream progress
- **🆕 Delivery dashboard:** [`roadmap/delivery-dashboard.md`](./roadmap/delivery-dashboard.md) — **executive snapshot + burndown + DORA + SLI/SLO + cost savings (auto-update hourly)**
- **Milestones backlog:** [`milestones/backlog/`](./milestones/backlog/) — 26 milestones MS-023a..x individuais
- **Milestones active:** [`milestones/active/`](./milestones/active/) — em progresso (max 2)
- **Milestones delivered:** [`milestones/delivered/`](./milestones/delivered/) — completados

## Pattern Catalog (Master)

ExchangeOS adiciona **20 catalogos** de patterns ao Allenty (continua a numeracao apos LedgerOS GP-/RP-/AU-/AP-/CP-/KP-/FP-/DS-):

| Catalog | Quantity | Path |
|---------|----------|------|
| FX-GP-* (Golang) | 40 | [`01-architecture/patterns/200-fx-golang-patterns.md`](./01-architecture/patterns/200-fx-golang-patterns.md) |
| FX-DDD-* (Domain-Driven Design) | 35 | [`01-architecture/patterns/201-fx-ddd-patterns.md`](./01-architecture/patterns/201-fx-ddd-patterns.md) |
| FX-EDA-* (Event-Driven Architecture) | 45 | [`01-architecture/patterns/202-fx-eda-patterns.md`](./01-architecture/patterns/202-fx-eda-patterns.md) |
| FX-CP-* (CockroachDB) | 50 | [`01-architecture/patterns/205-fx-cockroachdb-patterns.md`](./01-architecture/patterns/205-fx-cockroachdb-patterns.md) |
| FX-KP-* (Kafka) | 60 | [`01-architecture/patterns/206-fx-kafka-patterns.md`](./01-architecture/patterns/206-fx-kafka-patterns.md) |
| FX-FP-* (Apache Flink) | 40 | [`01-architecture/patterns/207-fx-flink-patterns.md`](./01-architecture/patterns/207-fx-flink-patterns.md) |
| FX-DS-* (DevSecOps & CI-CD) | 50 | [`01-architecture/patterns/210-fx-devsecops-cicd-patterns.md`](./01-architecture/patterns/210-fx-devsecops-cicd-patterns.md) |
| FX-K8S-* (Kubernetes) | 40 | [`01-architecture/patterns/211-fx-kubernetes-patterns.md`](./01-architecture/patterns/211-fx-kubernetes-patterns.md) |
| FX-IAC-* (Terraform + GCP) | 40 | [`01-architecture/patterns/212-fx-terraform-gcp-patterns.md`](./01-architecture/patterns/212-fx-terraform-gcp-patterns.md) |
| FX-DOC-* (Docker & Container) | 20 | [`01-architecture/patterns/213-fx-docker-container-patterns.md`](./01-architecture/patterns/213-fx-docker-container-patterns.md) |
| FX-GRPC-* (gRPC) | 55 | [`01-architecture/patterns/220-fx-grpc-patterns.md`](./01-architecture/patterns/220-fx-grpc-patterns.md) |
| FX-API-* (REST/OpenAPI/CRUD) | 50 | [`01-architecture/patterns/221-fx-api-rest-patterns.md`](./01-architecture/patterns/221-fx-api-rest-patterns.md) |
| FX-ASYNC-* (AsyncAPI 3.0) | 45 | [`01-architecture/patterns/222-fx-asyncapi-patterns.md`](./01-architecture/patterns/222-fx-asyncapi-patterns.md) |
| FX-IAM-* (IAM + RBAC + ISO 27001) | 50 | [`01-architecture/patterns/230-fx-iam-rbac-patterns.md`](./01-architecture/patterns/230-fx-iam-rbac-patterns.md) |
| FX-OTEL-* (OpenTelemetry Go) | 60 | [`01-architecture/patterns/240-fx-opentelemetry-patterns.md`](./01-architecture/patterns/240-fx-opentelemetry-patterns.md) |
| FX-TEST-* (Testing strategy) | 40 | [`01-architecture/patterns/250-fx-testing-patterns.md`](./01-architecture/patterns/250-fx-testing-patterns.md) |
| FX-QA-* (TDD + E2E + Security Local Gates) | 35 | [`01-architecture/patterns/260-fx-qa-tdd-e2e-patterns.md`](./01-architecture/patterns/260-fx-qa-tdd-e2e-patterns.md) |
| FX-SYNC-* (Database Sync + Cross-Module) | 40 | [`01-architecture/patterns/270-fx-sync-cross-module-patterns.md`](./01-architecture/patterns/270-fx-sync-cross-module-patterns.md) |
| FX-INT-* (Integration Verification) | 25 | [`01-architecture/patterns/280-fx-integration-verification-patterns.md`](./01-architecture/patterns/280-fx-integration-verification-patterns.md) |
| FX-XOS-* (Cross-Platform Tooling) | 20 | [`01-architecture/patterns/290-fx-cross-platform-patterns.md`](./01-architecture/patterns/290-fx-cross-platform-patterns.md) |
| FX-COMMIT-* (Pre-Commit HARD Enforcement) | 25 | [`01-architecture/patterns/300-fx-precommit-enforcement-patterns.md`](./01-architecture/patterns/300-fx-precommit-enforcement-patterns.md) |
| **TOTAL** | **850 patterns** | [`01-architecture/patterns/index.md`](./01-architecture/patterns/index.md) |

## Foco em Standards

| Standard | Coverage % | Workstream |
|----------|-----------|-----------|
| **ISO 20022 FX (`fxtr`)** | 100% (15 messages: 008/013/014/015/016/017/030/031-038) | 02 + 03 + 05 |
| **CLS Bank Protocol** | 100% (settlement member + PayIn cycle + NetReport) | 05 |
| **CFETS PTPP** | 100% (8 messages 031-038) | 05 |
| **BACEN Cambio** | 100% (Lei 14.286 + 8 Resolucoes + Circulares + IOF + VASP + eFX) | 09 |
| **ISO 27001:2022** | 93 Annex A controls mapeados (cert target Sprint 16) | 08 |
| **ISO 27000/27002/27003/27004/27005** | 100% framework + risk + metrics | 08 |
| **FIBO** | ≥ 80% das classes FX-relevantes referenciadas | 03 |
| **SLSA Level** | L3 obrigatorio | 07 |

## Open Questions Status

- **108 open questions** acumuladas (ver [`00-governance/open-questions.md`](./00-governance/open-questions.md))
- Prioridade critica: 3c (categoria licenca BACEN), 4a (cloud provider GCP), 9a (Lefthook vs pre-commit), 10a (shared hub TLS day-1)

## Sources

- **Pattern de referencia:** [LedgerOS .base/plans/](../../../ledgeros/.base/plans/) — espelhado 100%
- **Snapshot monolitico v3.11.7:** [`_archive/allenty-v3.11.7-monolithic-plan.md`](./_archive/allenty-v3.11.7-monolithic-plan.md) — 10.499 linhas preservadas como referencia historica
- **README + version + CHANGELOG:** ver arquivos no raiz

## Next Steps

1. Revisar [`roadmap/master-plan.md`](./roadmap/master-plan.md)
2. Aprovar open questions criticas
3. Iniciar **MS-023a: Foundation & Scaffolding** (Fase F1, Sprint 1)

# ExchangeOS — CLAUDE.md (Livro de Regras do Projeto)

> **Modulo:** ExchangeOS — Standalone FX Module (Revenu Platform)
> **Ports:** `:8084 HTTP / :9084 gRPC`
> **Stack:** Go 1.25 + CockroachDB + Kafka + Flink + GCP + ISO 20022 + IdentityOS/KeycloakOS

## Modular Memory Imports

> CLAUDE.md usa `@path/to/file` imports para load modular de memoria. Loaded automaticamente.

@.claude/context/glossary.md
@.claude/context/architecture-overview.md
@.claude/context/business-rules.md
@.claude/PERFORMANCE.md

## Visao do Projeto

ExchangeOS implementa o **Foreign Exchange (FX) Trading & Settlement** completo da Revenu Platform:
- ISO 20022 FX Business Domain (`fxtr` CLS + CFETS — 15 messages)
- BACEN Cambio 100% (Lei 14.286/2021 + Resolucoes BCB 277-561)
- Pricing CIP nativo (iotafinance formula)
- Integration nativa com 13 modulos da plataforma
- ISO 27001:2022 certification target

## Regras Criticas (NUNCA negociaveis)

### Money & Precisao
- **NUNCA `float64` para money/rate.** Sempre `decimal.Decimal` (shopspring/decimal)
- NUMERIC(20,8) interno para rates, NUMERIC(20,4) display, NUMERIC(20,2) money (excecoes JPY=0, BHD=3)
- Banker's rounding (half-even) para evitar bias

### ISO 20022
- **`fxti` e `fxmt` NAO existem** na ISO 20022 oficial — apenas `fxtr` (15 messages)
- Quote/Amendment vivem como gRPC interno; traduzem para fxtr.014/015/016 (CLS) ou fxtr.031/035/036 (CFETS) na fronteira

### Database
- **Shared CRDB hub TLS** desde dia 1 (`cockroachdb/modules/exchangeos/`)
- NUNCA inline `--insecure` (como accountos/paymentos legacy)
- Tenant SoT em AccountOS (FK conceitual + CDC materialized view)

### Security
- mTLS para TODAS as conexoes inter-service
- TLS 1.3 minimum
- OAuth2 client_credentials (RFC 6749 4.4) com client_secret em Vault (NUNCA em codigo)
- 93 ISO 27001 Annex A controls cobertos

### Pre-Commit
- **`--no-verify` BLOQUEADO** via `scripts/git-hooks-wrapper.sh`
- 3 tiers SLO: pre-commit < 30s / pre-push < 3min / pre-merge < 15min
- Coverage gate: domain >= 80%, application >= 70%

### Cross-Platform
- Task (taskfile.dev) source-of-truth; Makefile auto-gerado
- Roda identico em macOS, Linux, Windows, WSL2, Alpine

## Architecture-as-Code Standards

| Layer | Source of Truth |
|-------|----------------|
| Domain rules | `modules/<bc>/domain/specifications/` (50 RN_FX_*) |
| Ontology | `.base/aasc/ontology/core/<bc>.ttl` (OWL 2 DL + SHACL) |
| ERDs | `.base/erds/domain/erd-<bc>-domain.md` |
| Flows | `.base/flows/<subdomain>/RFLW.024.NNN.NN.md` (Mermaid) |
| Patterns | `.base/plans/01-architecture/patterns/NNN-fx-<topic>-patterns.md` |
| API contracts | `proto/exchangeos/v1/*.proto`, `api/openapi/exchangeos-v1.yaml`, `api/asyncapi/exchangeos-v1.yaml` |
| Migrations | `migrations/000NNN_*.up.sql` + `*.down.sql` |
| Plans | `.base/plans/` (12 workstreams + 26 milestones) |

## Entry Points

- **Plan master:** `.base/plans/index.md`
- **Roadmap:** `.base/plans/roadmap/master-plan.md`
- **Milestones:** `.base/plans/milestones/backlog/`
- **Monolithic snapshot:** `.base/plans/_archive/allenty-v3.11.7-monolithic-plan.md`

## Workflow Padronizado

1. **TDD obrigatorio** em `modules/<bc>/domain/` e `pkg/pricing/` (Red-Green-Refactor)
2. **Conventional Commits** (`feat(scope):`, `fix(scope):`, `docs(scope):`, etc)
3. **Branch protection:** 2 approvers + signed commits + status checks all green
4. **Local gates** rodam antes de qualquer push (lefthook)
5. **CI mirrors local** (zero discrepancia)

## Agents (subagents disponiveis em `.claude/agents/`)

Para tarefas complexas, **delegue para subagents especializados em paralelo**:

| Agent | Quando usar |
|-------|-------------|
| `fx-domain` | Modelagem de domain (aggregates, VOs, specs) — TDD-first |
| `iso20022` | XSD parsing, message marshaling, BAH handling |
| `bacen-compliance` | DEC, SCE-*, IOF, classificacao 95 codigos, SISCOAF |
| `pricing-quant` | CIP formula, NDF, PTAX, cross-rate, MTM |
| `cls-settlement` | CLS PvP, PayIn cycle, NetReport, fxtr CLS variants |
| `cfets-confirmation` | CFETS PTPP, fxtr 031-038 |
| `ontology-shacl` | TTL v1.2.0, SHACL validation, FIBO alignment |
| `database-crdb` | CockroachDB schemas, migrations, CDC, multi-CCY postings |
| `kafka-flink` | Kafka topics, ACLs, Flink jobs, EDA patterns |
| `iam-security` | IdentityOS, KeycloakOS, RBAC, ISO 27001 |
| `observability-otel` | OTel instrumentation, dashboards, SLI/SLO |
| `testing-qa` | TDD, E2E, CRUD tests, security gates |
| `devsecops-cicd` | GitHub Actions, SLSA L3, supply chain, Lefthook |
| `infra-k8s-terraform` | GKE Autopilot, Terraform GCP, Helm, Vault |
| `cross-platform` | Task, Makefile, PowerShell, bash POSIX |

Ver [`.claude/agents/`](./.claude/agents/) para definicoes completas.

## Skills (em `.claude/skills/`)

Skills sao **comandos especializados** invocaveis via slash command:

- `/fx-trade-book` — Book a new FX trade end-to-end
- `/fx-pricing-test` — Run pricing golden test cases (BIS/CME/PTAX)
- `/bacen-compliance-check` — Validate operation against BACEN rules
- `/ontology-validate` — Run SHACL validation on `.base/aasc/ontology/`
- `/integration-audit` — Run 4-vector × 13-module integration audit
- `/cost-savings-report` — Generate weekly GitHub Actions cost savings report

## Quando Pedir Ajuda

- Domain modeling dubio: invoke `fx-domain` agent
- Regulatory compliance unclear: invoke `bacen-compliance` agent
- Cross-cutting (security + observability + data): invoke MULTIPLE agents em paralelo

## Comunicacao

- Idioma padrao: **portugues** para contexto Brasil (BACEN, FX market locais), **ingles** para code/comments/commits
- Pre-commit em ingles (Conventional Commits convention)

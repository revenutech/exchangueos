# `.claude/agents/` — 15 Agentes Especializados Paralelos

> **Filosofia:** Tarefas cross-cutting sao decompostas e despachadas para **multiplos agentes em paralelo** (no message handling, isolated contexts, focused expertise).

## Quando Delegar para Agentes

| Sintoma | Acao |
|---------|------|
| Task tem multiplas dimensoes (domain + DB + Kafka + tests) | **Spawn agentes em paralelo** |
| Task profundamente especializada (ex: SHACL validation, fxtr.014 marshaling) | Delegate ao agent especifico |
| Task simples e localizada | Trabalhe direto (no delegation overhead) |
| Audit cross-cutting (security, integration) | Multi-agent paralelo + consolidacao |

## Catalogo (15 Agentes)

### Domain Agents

| Agent | Foco | Quando invocar |
|-------|------|----------------|
| **`fx-domain`** | DDD modeling — aggregates, VOs, services, specs | Toda mudanca em `modules/<bc>/domain/` |
| **`pricing-quant`** | Pricing math — CIP, NDF, PTAX, MTM, cross-rate | Mudancas em `pkg/pricing/` ou modelagem quant |

### ISO 20022 & Standards Agents

| Agent | Foco | Quando invocar |
|-------|------|----------------|
| **`iso20022`** | XSD parsing, message marshaling, BAH | Trabalhar com `pkg/iso20022/`, novos schemas |
| **`cls-settlement`** | CLS PvP, PayIn cycle camt.061/062/063, NetReport camt.088, fxtr CLS variants | CLS-related work |
| **`cfets-confirmation`** | CFETS PTPP, fxtr 031/032/033/034/035/036/037/038 | CFETS-related work |

### Compliance Agents

| Agent | Foco | Quando invocar |
|-------|------|----------------|
| **`bacen-compliance`** | DEC, SCE-IED, SCE-Credito, SCE-CBE, IOF, classificacao 95 codigos, SISCOAF, VASP, eFX | BACEN regulatory work |
| **`iam-security`** | IdentityOS integration, KeycloakOS realm + clients, ISO 27001 controls, OAuth2 client_credentials | IAM/security work |

### Data Layer Agents

| Agent | Foco | Quando invocar |
|-------|------|----------------|
| **`database-crdb`** | CockroachDB schemas, migrations, CDC CHANGEFEED, multi-CCY postings PvP, SHARED hub TLS | DB schema/migration work |
| **`kafka-flink`** | Kafka topics, ACLs, Schema Registry, Flink stateful jobs, CEP fraud | Streaming/messaging work |
| **`ontology-shacl`** | TTL v1.2.0, OWL 2 DL, SHACL, FIBO alignment, semantic conventions | Ontology work em `.base/aasc/ontology/` |

### Operations Agents

| Agent | Foco | Quando invocar |
|-------|------|----------------|
| **`observability-otel`** | OTel instrumentation, Tempo/Mimir/Loki, Grafana dashboards, SLI/SLO | Observability/telemetry |
| **`testing-qa`** | TDD Red-Green-Refactor, CRUD tests, E2E, security gates, FX-TEST/FX-QA patterns | Test design + implementation |
| **`devsecops-cicd`** | GitHub Actions workflows, SLSA L3, Cosign, SBOM, supply chain, Lefthook | CI/CD work |
| **`infra-k8s-terraform`** | GKE Autopilot, Helm charts, Terraform GCP, Vault, KMS, VPC SC, Binary Auth | Infrastructure work |
| **`cross-platform`** | Task (taskfile.dev), Makefile auto-gen, PowerShell mirror, bash POSIX, docker cross-platform | Cross-platform tooling |

## Parallel Invocation Pattern

```
User: "Implementar PayIn ACK end-to-end"

Orchestrator decomposes:
  ├─ fx-domain        → modela PayInACK aggregate + state machine
  ├─ iso20022         → marshaling camt.063.001.02
  ├─ database-crdb    → migration `payin_acks` table + ERD update
  ├─ kafka-flink      → publish event `exchangeos.payin.acked`
  ├─ bacen-compliance → validate BACEN rules (se aplicavel)
  ├─ observability-otel → spans + metrics + log
  └─ testing-qa       → TDD tests + CRUD + integration

All execute in parallel → orchestrator consolida → user receives coherent diff
```

## Agent Definition Template

Cada agente em `.claude/agents/<name>.md` segue:

```markdown
---
name: <agent-name>
description: <one-line description>
tools: [Read, Edit, Write, Bash, Grep, ...]
model: opus  # or sonnet
---

# Agent: <Name>

## Mission
<2-3 paragraphs sobre area de expertise>

## Core Files & Paths
- <paths que o agent monitora/edita>

## Conventions & Rules
- <regras especificas>

## Workflows
- <fluxos comuns que executa>

## Anti-Patterns (NUNCA fazer)
- <coisas a evitar>

## Cross-References
- <related agents, docs, patterns>
```

## Como Criar Novo Agent

1. `.claude/agents/<new-name>.md` com template acima
2. Update este `index.md`
3. Update `CLAUDE.md` raiz (se relevante para todo o projeto)
4. Pode ser invocado via task tool ou auto-spawn pelo orchestrator

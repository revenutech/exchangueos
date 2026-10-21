# ExchangeOS — Monorepo INDEX

> **Mapa do monorepo ExchangeOS** — entry point para navegacao.

## Root Files

| File | Purpose |
|------|---------|
| `CLAUDE.md` | Livro de regras do projeto (shared) |
| `CLAUDE.local.md` | Overrides pessoais (gitignored) |
| `INDEX.md` | Este arquivo — mapa monorepo |
| `Taskfile.yml` | Source-of-truth de build/test/run (cross-platform) |
| `Makefile` | Backward compat (auto-gerado de Taskfile) |
| `lefthook.yml` | Git hooks (pre-commit + pre-push + pre-merge) |
| `.pre-commit-config.yaml` | Fallback pre-commit Python |
| `.gitattributes` | Line endings (LF/CRLF per file type) |
| `.gitignore` | Standard + IDE + Go + secrets |
| `go.mod` | `github.com/revenutech/exchangeos` |
| `docker-compose.local.yml` | Local stack (app + dependencias) |
| `docker-compose.deps.yml` | Dependencias auxiliares (Kafka + Vault + Keycloak + OTel) |
| `docker-compose.test.yml` | CI testcontainers stack |
| `Dockerfile` + `Dockerfile.dev` + `Dockerfile.test` | Multi-stage distroless builds |

## Top-Level Folders

| Folder | Purpose |
|--------|---------|
| `.base/` | Architecture-as-Code (AasC) — plans, ontology, flows, erds |
| `.claude/` | Claude Code configuration (hooks, agents, skills, rules) |
| `.github/` | GitHub Actions workflows + branch protection + PR template |
| `cmd/` | Entrypoints: api, worker, cls-cycle, eod, mq-bridge, cred-rotator, migrator |
| `modules/` | 14 Bounded Contexts (trade, quote, amendment, cls_settlement, payin, netreport, cfets_capture, cfets_confirmation, settlement, refdata, admin, risk, position, compliance) |
| `pkg/` | Shared packages (pricing, iso20022, ledger, iam, telemetry, integration, mq, events, outbox, resilience, tenant, health) |
| `internal/` | Internal config (db, kafka, ibmmq, vault, telemetry, middleware, tls) |
| `proto/` | Protobuf contracts (`proto/exchangeos/v1/`) |
| `api/` | OpenAPI 3.1 + AsyncAPI 3.0 + Postman + HTML docs |
| `migrations/` | CockroachDB DDL (000001-000020+) |
| `seeds/` | Seed SQL files (tenants, currency_pairs, calendars, SSI, etc) |
| `tests/` | Test suite (unit, integration, e2e, contract, load, compliance, patterns) |
| `k8s/` | Helm charts + Kustomize overlays + NetworkPolicies + OPA |
| `infra/` | Terraform (GCP modules + environments) |
| `scripts/` | Bash POSIX scripts + `scripts/win/` PowerShell mirror |
| `docs/` | Markdown docs (onboarding, troubleshooting, runbooks) |
| `specs/` | Spec-Driven Development specs (feature contracts) |
| `bin/` | Built binaries (gitignored) |
| `certs/` | Symlink para `cockroachdb/modules/exchangeos/certs/` |

## .base/ (Architecture-as-Code)

| Subfolder | Purpose |
|-----------|---------|
| `.base/plans/` | 12 workstreams + 26 milestones + roadmap + versionamento SemVer |
| `.base/aasc/ontology/` | 35 TTL v1.2.0 (core + bridges + shapes + compliance + domains) |
| `.base/flows/` | 85 flows individuais RFLW.024.NNN.NN |
| `.base/erds/` | 23 ERDs (14 BC + 5 cross-BC + 4 common) + 16 SQL DDL |
| `.base/docs/` | Architecture docs auxiliares |

## .claude/ (Claude Code Configuration)

| Subfolder | Purpose |
|-----------|---------|
| `.claude/agents/` | 15 agentes especializados (fx-domain, iso20022, bacen-compliance, etc) — paralelo |
| `.claude/skills/` | 6+ skills slash commands (/fx-trade-book, /ontology-validate, etc) |
| `.claude/hooks/` | Determinastic hooks (pre-push, on-mcp-call) |
| `.claude/rules/` | Path-scoped rules (glob match) |
| `.claude/output-styles/` | Custom response formats |
| `.claude/commands/` | Slash commands (legacy LedgerOS-pattern) |
| `.claude/settings.json` | Allowed permissions (shared) |
| `.claude/settings.local.json` | Personal permissions (gitignored) |

## Quick Start

```bash
# 1. Onboarding (qualquer SO)
make install-hooks
make install-tools

# 2. Subir local stack
make local-up

# 3. Run tests
make tdd          # TDD watch mode
make test         # unit tests
make test-crud    # CRUD integration

# 4. Antes do commit (automatico via lefthook)
git commit -m "feat(trade): ..."   # pre-commit hook < 30s

# 5. Antes do push (automatico)
git push   # pre-push hook < 3min

# 6. Antes do PR
make premerge   # full pre-merge gates < 15min
gh pr create
```

## Cross-References

- **Master plan:** [`.base/plans/index.md`](./.base/plans/index.md)
- **Roadmap:** [`.base/plans/roadmap/master-plan.md`](./.base/plans/roadmap/master-plan.md)
- **Milestones:** [`.base/plans/milestones/`](./.base/plans/milestones/)
- **Project rules:** [`CLAUDE.md`](./CLAUDE.md)
- **Claude agents:** [`.claude/agents/index.md`](./.claude/agents/index.md)
- **Claude skills:** [`.claude/skills/index.md`](./.claude/skills/index.md)

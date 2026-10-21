# ExchangeOS Engineering Onboarding

> Welcome. Plan v4.18.0+. 14 bounded contexts. 9 migrations. 27 documented patterns. 13 ADRs (initial 8 + future).
> If you're new, work through this end-to-end on your first day.

## Day 1 — local setup + first PR

### Prereqs

- Go 1.25.1+ (`go version`)
- Docker + docker compose
- [Task](https://taskfile.dev) (`brew install go-task/tap/go-task` on mac)
- [Lefthook](https://github.com/evilmartians/lefthook) (`brew install lefthook`)
- `gh` CLI authenticated to `revenu-tech` org

### First clone

```bash
git clone git@github.com:revenu-tech/exchangeos.git
cd exchangeos

# Install dev toolchain (buf, lefthook, golangci-lint, protoc plugins).
task install

# Bring up the local stack — CRDB + Kafka + OTel + exchangeos-api.
task compose:up

# Verify the smoke endpoint.
curl http://localhost:8084/v1/refdata/currencies?active_only=true | jq .count
```

### First test run

```bash
task test            # ~271 unit tests; expected < 60s on M1 mac
task test:e2e        # 4 E2E scenarios against the local stack
```

### First PR

Goal: surface yourself in `git log`.

1. Read [CLAUDE.md](../../CLAUDE.md) — project rules. Non-negotiables.
2. Pick a "good first issue" labelled PR — typically a missing test case or doc typo.
3. Branch: `git checkout -b yourname/first-pr`
4. Make change → `task hooks:pre-commit` will run (HARD enforcement; see ADR 0004).
5. Push → CI runs full suite + lint + security scans.
6. Request review from 2 platform engineers.
7. Merge on green.

**If lefthook blocks you:** read the error carefully. It's probably right. NEVER use `--no-verify` (the wrapper script blocks it). Emergency bypass requires CISO sign-off (see SoD matrix).

## Day 7 — repo tour

Spend 30 min in each section:

| Folder | What lives here | Read this file first |
|--------|-----------------|---------------------|
| `.base/plans/` | Master plan, 26 milestones (all delivered), CHANGELOG | `.base/plans/index.md` |
| `.base/aasc/ontology/core/` | 9 OWL 2 DL ontologies | `trade.ttl` |
| `.base/flows/` | 8 Mermaid sequence diagrams | `trade/RFLW.024.001.01.md` |
| `.base/erds/` | 5 Mermaid erDiagram + sync rule | `erd-trade-domain.md` |
| `.base/plans/01-architecture/patterns/` | 6 pattern catalogs (27 docs) | `README.md` |
| `cmd/` | 7 binaries (api/worker/migrator/cls-cycle/eod/mq-bridge/cred-rotator) | `cmd/api/main.go` |
| `modules/` | 14 bounded contexts × {domain, application, infrastructure, api} | `modules/trade/` (canonical template) |
| `pkg/` | Shared libs: pricing, iso20022, bacen, outbox, health | `pkg/pricing/doc.go` |
| `internal/` | Cross-cutting: config, container, db, eventbus, telemetry | `internal/container/container.go` |
| `migrations/` | 9 numbered SQL pairs | `migrations/README.md` |
| `deploy/` | Helm + Terraform + Argo Rollouts + cert-manager + Kafka topics | `deploy/helm/exchangeos/Chart.yaml` |
| `docs/operations/` | Runbooks: go-live, canary, performance, CRDB hub PR | `docs/operations/runbook-index.md` |
| `docs/security/` | ISO 27001 mapping + threat model + SoD + IR + DR | `docs/security/iso27001-controls-mapping.md` |
| `docs/integrations/` | Sibling module contracts | `docs/integrations/README.md` |
| `docs/adr/` | Architecture Decision Records (8+) | `docs/adr/README.md` |

## Day 30 — own a bounded context

Goal: be able to make changes to any BC without supervision.

- [ ] Pair with current owner on a real ticket in one BC
- [ ] Walk through the BC's domain.go, application/service.go, repos.go, api/grpc_server.go
- [ ] Run the BC's test suite + add a test for an uncovered branch
- [ ] Trace one event end-to-end via OTel (Quote → Trade → Settlement)
- [ ] Read the BC's ADR if applicable + the corresponding RFLW flow + ERD

## Day 90 — full member

Goal: drive design decisions + on-call duty.

- [ ] First on-call rotation week (shadowed first time, solo after)
- [ ] Run one tabletop drill as IC (see `docs/security/drills/template.md`)
- [ ] Author one ADR (any decision worth recording)
- [ ] Lead one production canary deploy (`task canary:*` + `docs/operations/canary-runbook.md`)

## Key conventions (cited from CLAUDE.md)

- **NEVER `float64` for money/rate.** Lint-enforced. See ADR 0005.
- **NEVER `--insecure` CRDB in non-local code.** See ADR 0001.
- **NEVER `git commit --no-verify`.** See FX-DS-001.
- **Conventional Commits** — `feat(scope): ...` / `fix(scope): ...` / `docs(scope): ...`. commit-msg hook enforces.
- **TDD** in domain/ layer (Red-Green-Refactor).
- **Aggregate constructors return `(*T, error)`.** See FX-GP-001 + ADR 0002.

## Where to ask questions

| Question type | Where |
|---------------|-------|
| "How do I do X?" | Slack `#exchangeos` |
| "Is X broken?" | Slack `#exchangeos-incidents` (read-only unless escalating) |
| "Should we do X?" | Open an ADR draft PR; tag for review |
| "Why is X this way?" | Search `.base/plans/CHANGELOG.md` + `docs/adr/` |

## What "good" looks like

- PR opens with a clear problem statement + a 2-3 line summary of approach
- Tests added/updated; coverage doesn't regress
- Public-API changes documented + ADR if architectural
- Conventional Commits subject under 70 chars
- CI green on first push (run `task hooks:pre-push` locally)
- Reviewers thank you for making their review easy

Welcome aboard. 🎉

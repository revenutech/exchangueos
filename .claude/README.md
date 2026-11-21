# `.claude/` — ExchangeOS Claude Code Configuration

> **Onde o Claude Code procura primeiro** ao trabalhar neste projeto.
> Reference: [Claude Code docs](https://docs.claude.com/en/docs/claude-code).

## Estrutura Completa (Best Practices 2026)

```
.claude/
├── README.md                    # Este arquivo
├── PERFORMANCE.md               # Parallel batching + context optimization
├── NOTIFICATIONS.md             # Notification routing config
├── TELEMETRY.md                 # Usage tracking docs
├── .gitignore                   # Exclude cache, telemetry, local
│
├── settings.json                # Permissions + model + env + hooks + statusLine (SHARED)
├── settings.local.json          # Personal permissions (GITIGNORED)
│
├── agents/                      # 15 SUBAGENTS especializados parallel-ready
├── skills/                      # 6 SLASH COMMANDS especializados
├── commands/                    # 9 commands SHORT (status, test, build, lint, etc)
├── hooks/                       # 12 HOOKS lifecycle completo
├── rules/                       # 7 RULES path-scoped (glob match)
├── output-styles/               # 3 OUTPUT STYLES (writing, code-review, tdd)
│
├── scripts/                     # Helper scripts (statusline.sh, etc)
├── context/                     # KNOWLEDGE CACHE (glossary, architecture, business rules)
├── memory/                      # CROSS-SESSION CONTEXT (sessions log, agent runs, MCP knowledge graph)
├── cache/                       # PERFORMANCE CACHE (gitignored, auto-pruned 30d)
├── plugins/                     # PROJECT-SCOPED PLUGINS (roadmap)
└── specs/                       # SDD FEATURE CONTRACTS (template + active specs)
```

## Hooks Lifecycle Completo (12 hooks)

| Event | Hook | Purpose |
|-------|------|---------|
| `SessionStart` | `session-start.sh` | Load context + show project status |
| `UserPromptSubmit` | `user-prompt-submit.sh` | **Auto-inject context hints** (trade/CLS/BACEN/ontology) |
| `PreToolUse(Bash)` | `pre-bash.sh` | Audit + block destructive patterns + tips |
| `PreToolUse(Write\|Edit)` | `pre-write.sh` | Block writes em secrets/.env/.git/ |
| `PreToolUse(mcp__*)` | `on-mcp-call.sh` | MCP audit + validation |
| `PostToolUse(Write\|Edit)` | `on-file-save.sh` | Auto-format Go + lint TTL |
| `PostToolUse(Bash)` | `post-bash.sh` | Log failures + auto-suggest fixes |
| `Stop` | `session-stop.sh` | Session summary + cache prune check |
| `SubagentStop` | `subagent-stop.sh` | Log agent runs |
| `Notification` | `notification.sh` | Route to desktop (macOS) + log |
| `SessionEnd` | `session-end.sh` | Final cleanup + metrics |
| (manual) | `pre-push.sh` | Type-check + tests antes do push |

## Subagents Parallel Pattern

Para tarefas cross-cutting, spawn agents **em paralelo**:

```
/parallel "Implementar PayIn ACK end-to-end"

→ Spawns simultaneamente:
  - fx-domain (modelagem)
  - iso20022 (camt.063 marshaling)
  - database-crdb (migration)
  - kafka-flink (publish event)
  - bacen-compliance (validate)
  - observability-otel (spans)
  - testing-qa (TDD tests)

→ Consolida → diff coerente
```

15 subagents:
- Domain: `fx-domain`, `pricing-quant`
- Standards: `iso20022`, `cls-settlement`, `cfets-confirmation`
- Compliance: `bacen-compliance`, `iam-security`
- Data: `database-crdb`, `kafka-flink`, `ontology-shacl`
- Operations: `observability-otel`, `testing-qa`, `devsecops-cicd`, `infra-k8s-terraform`, `cross-platform`

## Slash Commands (15 total)

### Skills (workflows complexos com YAML frontmatter)
- `/fx-trade-book` — Book trade end-to-end multi-agent
- `/fx-pricing-test` — Golden tests CIP/NDF/cross-rate
- `/bacen-compliance-check` — Full BACEN validation
- `/ontology-validate` — SHACL + OWL 2 DL profile
- `/integration-audit` — 4-vector × 13-module matrix
- `/cost-savings-report` — Weekly cost reporting

### Commands (operacoes basicas)
- `/status` — Project health check
- `/test` [unit|integration|crud|e2e|all]
- `/build` — All binaries
- `/lint` — All linters
- `/security-scan` — SAST + SCA + secrets + container + IaC
- `/agent <name> <prompt>` — Invoke especifico
- `/parallel <task>` — Spawn multi-agents
- `/milestone <list|show|start|complete>`
- `/audit` — Integration audit completo

## Rules Path-Scoped (7 rules — auto-load via glob)

| Glob | Rule | Carrega quando |
|------|------|----------------|
| `modules/**/domain/**/*.go` | `modules-domain.md` | Edit domain layer |
| `pkg/pricing/**/*.go` | `pkg-pricing.md` | Edit pricing |
| `proto/**/*.proto` | `proto-contracts.md` | Edit proto |
| `migrations/*.sql` | `migrations.md` | Edit migration |
| `.base/aasc/ontology/**/*.ttl` | `ontology-ttl.md` | Edit TTL |
| `.base/flows/**/*.md` | `flows-mermaid.md` | Edit flow |
| `tests/**/*_test.go` | `tests.md` | Edit test |

## Output Styles

- `writing.md` — Default (concise, structured, portugues + ingles)
- `code-review.md` — Estrutura code review padronizada
- `tdd.md` — TDD-focused (Red/Green/Refactor structure)

## Context Cache (load via @import em CLAUDE.md)

Em CLAUDE.md raiz:
```markdown
@.claude/context/glossary.md
@.claude/context/architecture-overview.md
@.claude/context/business-rules.md
@.claude/PERFORMANCE.md
```

Loaded once per session, cached.

## Settings.json Highlights

| Field | Value |
|-------|-------|
| `model` | `claude-opus-4-7` |
| `cleanupPeriodDays` | `30` — auto-prune cache |
| `includeCoAuthoredBy` | `true` |
| `enableAllProjectMcpServers` | `true` — auto-load `.mcp.json` |
| `statusLine` | Custom via `.claude/scripts/statusline.sh` |
| `env` | Project-wide env vars (EXCHANGEOS_MODULE, LEFTHOOK_QUIET, etc) |
| `permissions.defaultMode` | `acceptEdits` (smart UX) |
| `permissions.additionalDirectories` | 7 sibling repos (cockroachdb, ledgeros, etc) |
| `permissions.allow` | 80+ Bash patterns + WebFetch + WebSearch |
| `permissions.deny` | 20+ destructive patterns (rm -rf, --no-verify, secrets) |
| `permissions.ask` | 9 patterns requering confirmation (terraform destroy, etc) |
| `hooks` | 8 hook events configurados (Session*, UserPromptSubmit, PreToolUse, PostToolUse, Stop, SubagentStop, Notification) |

## MCP Servers (.mcp.json)

6 MCP servers configurados:
- `filesystem` — Scoped filesystem access
- `github` — GitHub API (PRs, issues, workflows)
- `postgres-crdb` — CockroachDB queries (read-only)
- `sequential-thinking` — Step-by-step structured reasoning
- `fetch` — URL fetching
- `memory` — Knowledge graph persistente

## Cross-References

- **Project rules:** [`../CLAUDE.md`](../CLAUDE.md)
- **Monorepo INDEX:** [`../INDEX.md`](../INDEX.md)
- **Plan master:** [`../.base/plans/index.md`](../.base/plans/index.md)
- **Roadmap:** [`../.base/plans/roadmap/master-plan.md`](../.base/plans/roadmap/master-plan.md)

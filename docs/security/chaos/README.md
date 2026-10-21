# Chaos Engineering Program — ExchangeOS

> "Hope is not a strategy." — Sysadmin proverb
>
> We don't trust resilience claims that haven't been exercised in production-like
> environments. Each scenario below has a Litmus YAML manifest, a runbook, and
> a quarterly drill cadence. Cross-referenced from `docs/security/dr-runbook.md`.

## Principles

1. **Steady state first** — define the metric that means "the system is fine" before injecting
2. **Smallest blast radius** — start in `staging`, only graduate to `production` after 3 successful runs
3. **Stop conditions** — every experiment has automatic abort criteria
4. **Hypothesis-driven** — write expected behaviour before running; record actual vs expected
5. **Post-experiment ALWAYS** — even successful runs surface findings worth fixing

## Tooling

| Tool | Role |
|------|------|
| **Litmus (ChaosCenter)** | Experiment orchestration + scheduling (GA, CNCF Incubating) |
| **Chaos Mesh** | Alternative experiment runner — used for network primitives |
| **kubectl drain** | Manual node-level chaos (controlled) |
| **Toxiproxy** | Local-dev TCP layer fault injection |
| **k6** | Load + fault combined (`tests/load/k6-trade-book.js`) |

## Experiment catalogue (10 scenarios)

| ID | Scenario | Hypothesis | Steady-state SLI | Severity |
|----|----------|-----------|------------------|----------|
| CHAOS-01 | Kill exchangeos-api pod | HPA replaces in < 30s; no 5xx burst | 5xx rate < 1% during chaos | LOW |
| CHAOS-02 | Kill worker pod | Outbox dispatch resumes < 60s; no message loss | outbox_pending_oldest_seconds < 120s | MEDIUM |
| CHAOS-03 | Drain GKE node carrying api pods | Pods reschedule; canary metrics unaffected | p99 latency < 200ms throughout | MEDIUM |
| CHAOS-04 | Inject 200ms latency CRDB → api | Connection pool absorbs; p99 < 500ms | p99 trade booking < 500ms | MEDIUM |
| CHAOS-05 | Drop 10% packets api ↔ Kafka | Outbox marks failed + retries succeed; no duplicate publishes | outbox_failed_total stable < 0.1% | HIGH |
| CHAOS-06 | CRDB primary region failover | api fails-over to secondary in < RTO 4h | DR runbook validates | CRITICAL |
| CHAOS-07 | Vault unavailable for 5min | api uses cached secrets; no auth failures | error_rate < 1% | HIGH |
| CHAOS-08 | IdentityOS OIDC discovery 503 | Token verification uses cached JWKS; existing sessions unaffected | auth_5xx < 0.1% | HIGH |
| CHAOS-09 | PTAX feed (OLINDA) unreachable | api serves last-good PTAX (max 24h stale) + alerts ops | ptax_staleness_seconds < 86400 | MEDIUM |
| CHAOS-10 | Kafka broker (1 of 3) killed | Producer retries succeed; in-sync replicas keep min.isr | publish_success_rate > 99.9% | HIGH |

## Cadence

| Frequency | Scope |
|-----------|-------|
| **Weekly** (automated) | CHAOS-01 + CHAOS-02 in staging via Litmus cron |
| **Monthly** (manual) | One of CHAOS-03..05 in staging + post-experiment writeup |
| **Quarterly** (chaos day) | CHAOS-06 (DR failover) + CHAOS-07/08 (auth/secrets) in dedicated DR cluster |
| **Annual** | Full chaos day in production canary with all 10 |

## Roles

- **Game-day lead** — Platform engineer; drives the experiment
- **Observer(s)** — On-call engineer + 1 BC owner whose code is in scope
- **Note-taker** — Records timeline + decisions
- **Comms** — Posts updates to `#exchangeos-chaos` (out-of-band from `#exchangeos-incidents`)

## Manifests + runbooks

- `chaos-pod-kill.yaml` — Litmus pod-delete spec for CHAOS-01, CHAOS-02
- `chaos-network-latency.yaml` — Chaos Mesh NetworkChaos for CHAOS-04
- `chaos-network-loss.yaml` — Chaos Mesh NetworkChaos for CHAOS-05
- `chaos-day-runbook.md` — Step-by-step for the quarterly chaos day
- `experiment-template.md` — Post-experiment template (copy per run)

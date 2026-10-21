# ExchangeOS Runbook Index

> Single-page index of every runbook + playbook + operational checklist.
> Pin this to the on-call channel description.

## 🚀 Deployment & rollout

| Runbook | Purpose | When to use |
|---------|---------|-------------|
| [Go-Live Checklist](go-live-checklist.md) | T-24h pre-flight + sign-off | First production deploy ever |
| [Canary Runbook](canary-runbook.md) | 4-step canary promotion | Every release |
| [CRDB Hub TLS PR Spec](crdb-hub-tls-pr.md) | Cross-repo PR to register exchangeos in shared CRDB hub | Once, before first prod deploy |

## 🔐 Security & compliance

| Runbook | Purpose | When to use |
|---------|---------|-------------|
| [ISO 27001 Controls Mapping](../security/iso27001-controls-mapping.md) | 93 Annex A controls evidence | Quarterly review + audit cycle |
| [Threat Model (STRIDE + DREAD)](../security/threat-model-stride.md) | 15 threats scored | Every major release + post-incident |
| [SoD Matrix](../security/sod-matrix.md) | 7 roles × 23 actions allowance | Quarterly role review + audit |
| [Incident Response Playbook](../security/incident-response.md) | Sev1-4 response + 5 common scenarios | Page acknowledged |
| [DR Runbook](../security/dr-runbook.md) | RTO 4h / RPO 5min failover | Regional outage |
| [LGPD Erasure Workflow](../security/data-lifecycle/erasure-workflow.md) | 5-stage Art. 18 IV process (15-day SLA) — Stage 3 executor is `cmd/erasure-worker` | LGPD erasure request approved |

## 🛠️ Day 2 ops (Taskfile-driven)

```bash
task canary:status        # current rollout state
task canary:promote       # advance to next weight step (manual gate)
task canary:abort         # stop rollout, preserve stable
task canary:rollback      # restore previous stable revision
task smoke:prod           # full smoke validation (gates promotion)
task load:trade-book      # k6 sustained-load test
task dash-update          # refresh delivery dashboard
task db:migrate           # apply pending migrations
task db:reset CONFIRM=yes # destructive — last resort only
```

## 📊 Observability

- Grafana dashboards: `dashboard.observability/d/exchangeos-delivery` + `dashboard.observability/d/exchangeos-api-slo`
- Prometheus alerts: route via Alertmanager → PagerDuty service `exchangeos-api-prod` → Slack `#exchangeos-incidents`
- OTel traces: Tempo UI scoped to `service.name=exchangeos-*`
- Logs: Loki via Grafana, query `{namespace="exchangeos"}`

## 🔄 Communication channels

| Channel | Purpose |
|---------|---------|
| Slack `#exchangeos` | General development |
| Slack `#exchangeos-incidents` | All incident channels reference here |
| Slack `#vault-audit` | EMERGENCY_BYPASS + Vault policy changes |
| PagerDuty service `exchangeos-api-prod` | Sev1/2 alerts |
| Status page `status.exchangeos.revenu.tech` | Customer-facing status |

## 📚 Architecture references

- Master index: `.base/plans/index.md`
- All 26 milestones (delivered): `.base/plans/milestones/delivered/`
- ERDs: `.base/erds/domain/`
- Flow diagrams: `.base/flows/`
- Ontology: `.base/aasc/ontology/core/`
- Pattern catalogs: `.base/plans/01-architecture/patterns/`

## 🧪 Test surfaces

- Unit tests: `task test` (~271 tests + 1 benchmark)
- E2E tests: `task test:e2e` (requires `task compose:up`)
- Smoke (prod-like): `task smoke:prod` (set `EXCHANGEOS_BASE_URL`)
- Load: `task load:trade-book` (set `EXCHANGEOS_BASE_URL`; requires k6)
- Security scans: `task sec:secrets`, `task sec:trivy`, `task sec:govulncheck`

## 📝 Audit & retention

- `audit_events` table — 7 years retention (regulatory)
- `outbox_dispatched_archive` — 30 days retention
- `.git/audit-bypass.log` — emergency-bypass log; daily review by security team
- CRDB backups — 90-day retention (shared hub responsibility)

## 🔗 External dependencies

| Dep | Owner | Failure impact |
|-----|-------|----------------|
| Shared CRDB hub | Platform DBA | Critical — service unavailable |
| Vault | Security ops | Critical — secrets unreadable |
| Kafka cluster | Platform messaging | Degraded — outbox queues build up |
| KrakenD API gateway | Platform networking | Critical — entry point |
| IdentityOS + KeycloakOS | IAM team | Critical — auth fails |
| OTel Collector → Tempo/Mimir/Loki | Observability | Degraded — flying blind |
| GCP CloudDNS | Platform networking | Critical — DNS resolution fails |

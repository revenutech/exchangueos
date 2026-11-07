# MS-024m — Production Deployment (GKE + Litmus + Chaos Mesh + ArgoCD live)

| Field | Value |
|-------|-------|
| **Code** | MS-024m |
| **Name** | production-deployment |
| **Phase** | F-OPS-PROD |
| **Sprint** | 4 of MS-024 cycle (close-out milestone) |
| **Status** | BACKLOG |
| **Owner** | Platform |
| **Dependencies** | MS-024l (CRDB hub), MS-024c (creds), MS-024h (postgres repos), MS-024a/b (workers) — most of the cycle |

## Why this milestone

All deployment artefacts exist (Helm + Terraform GKE module + ArgoCD Application + Argo Rollouts canary + cert-manager + SLSA L3 + Vault seed + chaos manifests). **None have been applied to a real cluster.** Without this, the system is shelf-ware. This milestone closes the gap from "design done" to "running production with traffic".

## Description

Provision the production GCP project + GKE Autopilot cluster + supporting infra via Terraform, deploy via ArgoCD pull, run smoke + load + chaos against staging first, then promote to production via canary. Includes Litmus + Chaos Mesh install for the chaos program (MS-024k indirectly).

## Acceptance Criteria

### Infra (Terraform applied)
- [ ] `deploy/terraform/environments/production/` `terraform apply` succeeds
- [ ] GKE Autopilot cluster up (1.29+) with WIF enabled
- [ ] VPC + Cloud NAT + private cluster + master authorized networks restricted
- [ ] CMEK enabled on etcd via Cloud KMS HSM key
- [ ] Binary Authorization policy applied (only signed images)
- [ ] GCS archive bucket created with lifecycle policy
- [ ] Budgets module applied with production threshold

### Cluster bootstrap
- [ ] cert-manager installed + ClusterIssuer applied (LE prod + staging)
- [ ] ArgoCD installed + `application.yaml` + `AppProject` applied
- [ ] Argo Rollouts controller installed
- [ ] Litmus operator + Chaos Mesh installed in dedicated namespaces with RBAC restricted to chaos-runner SA
- [ ] External Secrets Operator installed + Vault auth/kubernetes bound
- [ ] Vault production cluster reachable + `scripts/vault-seed.sh` executed for prod path

### Workload
- [ ] ArgoCD syncs ExchangeOS Application — all pods reach Ready
- [ ] Helm release `exchangeos` deployed (api + worker + migrator job completes + cls-cycle + eod + mq-bridge + cred-rotator)
- [ ] Migrator job completed for migrations 000001-current
- [ ] All 14 BC services responding to gRPC + REST health probes
- [ ] Prometheus scraping all targets + Grafana dashboards (delivery + SLO + FinOps) showing data

### Validation
- [ ] `task smoke:prod` returns 0 (all 7 checks pass)
- [ ] `k6 run tests/load/k6-trade-book.js` against staging meets SLO thresholds
- [ ] One full canary rollout 10→30→60→100 from clean state with Prometheus AnalysisTemplate gates green
- [ ] CHAOS-01 + CHAOS-02 run successfully in staging via Litmus
- [ ] DR runbook restore-from-backup drill executed + RTO/RPO measured

### Operational handover
- [ ] On-call rotation enrolled in PagerDuty `exchangeos-api-prod`
- [ ] Runbook index links validated
- [ ] First chaos day scheduled
- [ ] First quarterly cost review scheduled
- [ ] First quarterly LGPD retention review scheduled
- [ ] First DR drill scheduled

## Deliverables

- Production GCP project state in Terraform Cloud workspace (or GCS backend) with applied resources
- ArgoCD UI URL + Slack screenshot of first successful sync
- Smoke + load + canary + chaos run reports archived in `.audit-bundles/prod-bringup-YYYYMMDD/`
- Updated `docs/operations/go-live-checklist.md` — all rows ✅ signed-off
- Public status page updated (if applicable)
- Engineering newsletter post about go-live

## Cross-References

- `docs/operations/go-live-checklist.md` — primary checklist this milestone closes
- `docs/operations/canary-runbook.md` — canary procedure
- `docs/security/dr-runbook.md` — restore drill
- `docs/security/chaos/chaos-day-runbook.md` — first chaos day kickoff
- All earlier MS-024* milestones (most are prerequisites)
- This milestone closes the "production-ready end-to-end" gap; success criterion = system serves real trades

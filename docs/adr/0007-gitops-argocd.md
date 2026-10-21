# ADR 0007 — GitOps via ArgoCD over imperative kubectl

- Status: Accepted
- Date: 2026-05-24

## Context

Production deploys need:

1. **Reproducibility** — every cluster state derivable from a git ref
2. **Audit trail** — who changed what, when, why
3. **Rollback** — fast revert to previous state without manual intervention
4. **Multi-cluster** — same chart deployable to dev/staging/production with values overrides

`kubectl apply -f` from a CI job covers (1) and partially (2), but (3) is manual and (4) error-prone.

## Decision

**ArgoCD pulls from this repo's `deploy/helm/exchangeos/` chart and reconciles the live cluster state.**

`deploy/argocd/application.yaml` defines:

- Source: `deploy/helm/exchangeos` + `values-production.yaml`
- Destination: `https://kubernetes.default.svc` namespace `exchangeos`
- Sync policy: `automated` with `prune: true` + `selfHeal: true`
- Sync options: `CreateNamespace=true`, `ServerSideApply=true`, `ApplyOutOfSyncOnly=true`
- Retry: 5× exponential backoff 30s → 5min

`AppProject revenu-platform` whitelists source repos (`github.com/revenutech/*`) + namespaces (`exchangeos*` + `observability`) + RBAC role `deployer` for the platform-team group.

## Consequences

### Positive

- **Reproducibility** — `git checkout v1.2.3` + `argocd app sync` recreates exact state
- **Self-healing** — out-of-band changes (`kubectl edit ...` by an operator) reconciled back to declared state
- **Easy rollback** — `git revert` + push triggers automatic sync to previous version
- **Multi-environment** — same chart, different values files
- **Audit trail** — every change is a git commit signed by an identifiable author

### Negative

- **Learning curve** — operators must shift from "apply this YAML" to "merge a PR"
- **Cluster-side ArgoCD operator** — one more component to operate
- **Out-of-band changes still possible** — selfHeal eventually reconciles them, but an attacker could create a brief drift window

### Mitigations

- Operator onboarding covered in `docs/onboarding/README.md`
- ArgoCD itself deployed via Helm + GitOps from a bootstrap repo (not chicken-and-egg in this module)
- RBAC denies most users `kubectl edit` capability on the `exchangeos` namespace

## Alternatives considered

- **Imperative kubectl** — fast but loses (3) and (4)
- **Flux** — equivalent feature set; chose ArgoCD for the UI maturity + AppProject RBAC model
- **Helmfile direct** — push-based deploy; loses self-healing

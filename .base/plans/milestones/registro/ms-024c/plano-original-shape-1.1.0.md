# MS-024c — Credential Rotator (real loop)

| Field | Value |
|-------|-------|
| **Code** | MS-024c |
| **Name** | cred-rotator |
| **Phase** | F-OPS-PROD |
| **Sprint** | 1 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | Security + Platform |
| **Dependencies** | Vault production deployment |

## Why this milestone

`cmd/cred-rotator/` has a skeleton + the 14-secret catalogue is documented in `docs/integrations/identityos.md`. The actual monthly rotation loop against Vault — generate new client_secret, push to Vault, push to IdentityOS/KeycloakOS, signal pods to reload via SIGHUP or rolling restart, alert on failure — is not implemented.

## Description

Build the real cron-driven rotator that orchestrates monthly OAuth2 `client_secret` rotation for the 14 M2M clients used by ExchangeOS + sibling integrations. Must be idempotent, observable, and capable of unattended operation under WIF identity.

## Acceptance Criteria

- [ ] `cmd/cred-rotator/main.go` driven by config `secrets-catalog.yaml` listing 14 M2M clients + their Vault paths + IdP backends
- [ ] Per-secret operation: generate 32-byte cryptorandom, update KeycloakOS via admin API, write to Vault, verify reachable from a sample read role, archive previous secret for 24h emergency rollback
- [ ] Rolling pod restart of consumers via `kubectl rollout restart` after successful rotation (configurable per-secret)
- [ ] Dry-run mode that does everything except the IdP write
- [ ] Locking via Vault `kv/locks/cred-rotator/<secret>` to prevent concurrent rotations
- [ ] Emits `audit_event(type='SECRET_ROTATED', secret_id, previous_hash, new_hash, rotated_at, actor='cred-rotator')`
- [ ] Slack notification to `#exchangeos-security` on success + page on failure
- [ ] Helm CronJob template `templates/cred-rotator-cronjob.yaml` scheduled monthly (1st of month, 03:00 UTC)
- [ ] Integration test against KeycloakOS testcontainer + Vault dev-server
- [ ] Metric `secret_rotation_last_success_timestamp{secret}` for alerting on miss
- [ ] Runbook entry covering emergency rollback (`scripts/cred-rollback.sh <secret>`)

## Deliverables

- `cmd/cred-rotator/main.go`
- `internal/credrotator/keycloak.go`
- `internal/credrotator/vault.go`
- `internal/credrotator/orchestrator.go` (lock + rotate + verify + restart)
- `config/secrets-catalog.yaml`
- `scripts/cred-rollback.sh`
- `deploy/helm/exchangeos/templates/cred-rotator-cronjob.yaml`
- `tests/integration/cred_rotator_test.go`

## Cross-References

- `docs/integrations/identityos.md` — 14-secret catalogue
- `docs/security/iso27001-controls-mapping.md` controls 5.16, 5.17, 8.5
- `docs/security/sod-matrix.md` — automated rotation actor role

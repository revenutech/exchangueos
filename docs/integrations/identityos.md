# Integration — ExchangeOS ↔ IdentityOS + KeycloakOS

> Owner: IAM team
> Compatible since: ExchangeOS v4.2.0 (config + env scaffolding)
> Status: ✅ Wired (env + Helm + Vault SPI integration in place)

## Purpose

IdentityOS is the Revenu Platform IAM control plane; KeycloakOS is the OIDC
identity provider. Every API request to ExchangeOS carries a JWT issued by
KeycloakOS, validated at the KrakenD API gateway upstream of ExchangeOS.
M2M client_secrets for 14 service identities are stored in Vault, rotated
monthly by `cmd/cred-rotator` via Vault SPI.

## Direction

```
KrakenD gateway ──▶ exchangeos-api (validated JWT in metadata)
                          │
                          │ (TenantContext extracted from claims)
                          ▼
                  application service layer

ExchangeOS cmd/cred-rotator (monthly cron) ──▶ KeycloakOS Admin API + Vault SPI
                                                  (rotate 14 M2M secrets)

KeycloakOS ── identity.actor_disabled.v1 ─▶ ExchangeOS
            ── identity.tenant_revoked.v1 ─▶
```

## Events ExchangeOS consumes

### `identity.actor_disabled.v1`

KeycloakOS disables an actor (user/M2M). ExchangeOS:

- Marks `actors.status = 'DISABLED'`
- Any in-flight requests with this actor's JWT continue until JWT expiry (5 min default)
- New requests rejected at gateway with 401

### `identity.tenant_revoked.v1`

Tenant access revoked at the platform level. ExchangeOS:

- Marks `tenants.status = 'SUSPENDED'`
- BookTrade rejects with `403 PERMISSION_DENIED`
- Existing trades continue to settle (kill-switch is access-level, not data-level)

## Sync RPCs (ExchangeOS → IdentityOS)

```protobuf
service IdentitySync {
  // Resolve the friendly name for an actor_sub (used in audit_events display).
  rpc ResolveActor(ResolveActorRequest) returns (ResolveActorResponse);
}
```

Cached for 1 hour in-process. Not on the hot path.

## JWT contract

KrakenD validates the JWT signature (RS256, keys via JWKS endpoint), then
forwards the following claims as gRPC metadata / HTTP headers:

| Claim | Header / Metadata key | Used for |
|-------|----------------------|----------|
| `sub` | `x-actor-sub` | actor_id resolution |
| `tid` | `x-tenant-id` | TenantContext.tenant_id |
| `scope` | `x-scope` | RBAC enforcement |
| `corr_id` | `x-correlation-id` | OTel trace correlation |

ExchangeOS NEVER validates JWT signature itself — that's the gateway's job.

## M2M client catalog (14 secrets)

Managed by `cmd/cred-rotator` (CronJob, monthly schedule `0 3 1 * *`):

| Client | Purpose | Vault path |
|--------|---------|------------|
| exchangeos-api | api ↔ KrakenD JWT signing | `secret/data/exchangeos/oidc` (this module) |
| exchangeos-trader-<tenant-N> × 10 | Per-tenant trader bot | `secret/data/exchangeos/m2m/trader-<N>` |
| exchangeos-eod | EOD CronJob worker | `secret/data/exchangeos/m2m/eod` |
| exchangeos-cls-cycle | CLS cycle scheduler | `secret/data/exchangeos/m2m/cls-cycle` |
| exchangeos-mq-bridge | SWIFT MT bridge | `secret/data/exchangeos/m2m/mq-bridge` |

Rotation runbook: each rotation emits OTel span + admin.system_event; rollback
within 24h requires re-fetching the previous version from Vault KV versioning.

## Failure semantics

- **KeycloakOS down:** existing JWTs continue working until expiry (5 min);
  new login flows fail. Cached actor resolutions serve from in-process cache.
- **Vault down:** new pods can't start (secrets unreadable at boot); running
  pods keep working. CRITICAL alert.
- **JWT signature invalid:** gateway returns 401; never reaches ExchangeOS.
- **Tenant suspended mid-request:** in-flight request completes; next request fails.

## Open questions

- [ ] JWT TTL: balance security (short TTL) vs UX (refresh storm)
- [ ] Tenant scoping on JWT: claim-based (`tid`) or per-client (one client per tenant)?
- [ ] M2M client_secret rotation grace period — 24h overlap window
- [ ] KeycloakOS realm structure: one realm per tenant, or one realm for the whole platform?

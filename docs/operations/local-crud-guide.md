# Local CRUD + Query Guide

> How to inspect + mutate every ExchangeOS table running locally. Two surfaces:
> the REST admin API (`/v1/admin/*`) for app-level operations, and direct CRDB
> SQL for ad-hoc / complex analytics.

## Pre-requisite

Stack up + admin API enabled:

```bash
docker compose -f docker/compose/docker-compose.yml up -d
# EXCHANGEOS_ENABLE_ADMIN_API=true is already set in the compose file.
```

Verify:

```bash
curl -s http://localhost:8084/v1/admin/_schemas | head -c 200
# → {"count":30,"schemas":[{...}]}
```

## Endpoint catalogue

| Verb | Path | Purpose |
|---|---|---|
| `GET` | `/v1/admin/_schemas` | Catalogue — 30 tables + their PKs + filters + mutability |
| `GET` | `/v1/admin/{table}` | LIST rows. Query params: `limit` (default 100, max 500), `offset`, plus per-table filters |
| `GET` | `/v1/admin/{table}/{id}` | GET row by primary key (+ optional tenant scope) |
| `POST` | `/v1/admin/{table}` | INSERT. JSON body uses schema column names. Unknown keys silently dropped (SQL-injection safe) |
| `PUT` | `/v1/admin/{table}/{id}` | UPDATE matching row. Body is partial; PK + unrecognised columns ignored |
| `DELETE` | `/v1/admin/{table}/{id}` | DELETE row (refdata + runtime). `audit-events` + `outbox-archive` are read-only → 405 |

### Tenant scoping

Tenant-scoped tables auto-filter by `tenant_id`. Default tenant = the dev one
(`00000000-0000-5000-8000-000000000001`). Override with header:

```bash
curl -H 'X-Tenant-Id: 00000000-0000-5000-8000-000000000003' \
  http://localhost:8084/v1/admin/fx-trades
```

### Allowlisted filters

Each table exposes a small set of filter columns. Anything else is silently
ignored. Inspect with:

```bash
curl -s http://localhost:8084/v1/admin/_schemas | jq '.schemas[] | {url, filters}'
```

## Examples

### LIST first 3 active currencies

```bash
curl -s 'http://localhost:8084/v1/admin/currencies?active=true&limit=3' | jq
```

```json
{
  "count": 3,
  "items": [
    {"code":"AUD","cls_eligible":true,"minor_units":2,...},
    {"code":"BHD","cls_eligible":false,"minor_units":3,...},
    ...
  ]
}
```

### LIST settled trades

```bash
curl -s 'http://localhost:8084/v1/admin/fx-trades?status=SETTLED' | jq
```

### GET specific trade

```bash
curl -s http://localhost:8084/v1/admin/fx-trades/33333333-0000-5000-8000-000000000001 | jq
```

### POST a new currency

```bash
curl -s -X POST -H 'Content-Type: application/json' \
  -d '{"code":"XTS","name":"Test","minor_units":2,"cls_eligible":false,"cfets_eligible":false,"active":true}' \
  http://localhost:8084/v1/admin/currencies
# → {"status":"created","table":"currencies"}
```

### PUT to flip status

```bash
curl -s -X PUT -H 'Content-Type: application/json' \
  -d '{"active":false}' \
  http://localhost:8084/v1/admin/currencies/XTS
# → {"status":"updated","table":"currencies","id":"XTS"}
```

### DELETE refdata

```bash
curl -s -X DELETE http://localhost:8084/v1/admin/currencies/XTS
# → {"status":"deleted","table":"currencies","id":"XTS"}
```

### Read-only tables

```bash
curl -s -X POST http://localhost:8084/v1/admin/audit-events
# → HTTP 405 — audit_events is read-only (regulatory)
```

## Direct CRDB access

For ad-hoc SQL (joins, aggregates, complex filters) the REST surface won't fit.
Use the `cockroach sql` CLI inside the running container:

```bash
docker exec -it exchangeos-crdb cockroach sql --insecure --host=crdb --database=exchangeos
```

Sample queries:

```sql
-- Trade volume by pair (last 30 days)
SELECT bought_currency || sold_currency AS pair, count(*) AS cnt, sum(bought_amount) AS notional
FROM fx_trades
WHERE trade_date > now() - INTERVAL '30 days'
GROUP BY pair ORDER BY cnt DESC;

-- CLS cycle health (last 5 cycles)
SELECT cycle_date, status,
       (SELECT count(*) FROM cls_cycle_trades cct WHERE cct.cycle_id = c.cycle_id) AS trade_count,
       closed_at - opened_at AS duration
FROM cls_cycles c
ORDER BY cycle_date DESC LIMIT 5;

-- Outbox health
SELECT 
  (SELECT count(*) FROM outbox_events WHERE dispatched_at IS NULL) AS pending,
  (SELECT count(*) FROM outbox_events WHERE dispatched_at IS NOT NULL) AS in_table_dispatched,
  (SELECT count(*) FROM outbox_dispatched_archive) AS in_archive,
  (SELECT count(*) FROM outbox_events WHERE attempt_count > 0 AND dispatched_at IS NULL) AS failed;

-- Risk: which limits are > 50% utilised?
SELECT limit_type, scope, cap, utilised, ROUND(utilised / cap * 100, 2) AS pct
FROM risk_limits
WHERE utilised >= cap * 0.5
ORDER BY pct DESC;

-- Compliance: count BACEN reports by status
SELECT report_type, status, count(*)
FROM bacen_reports
GROUP BY report_type, status
ORDER BY report_type, status;
```

## Smoke

`scripts/smoke-crud.sh` LISTs every exposed table and asserts `count >= MIN_ROWS`
(default 5):

```bash
bash scripts/smoke-crud.sh
# → ✅ All 30 tables satisfy minimum 5 rows.
```

CI / canary gate variant:

```bash
EXCHANGEOS_BASE_URL=https://api-staging.exchangeos.revenu.tech \
MIN_ROWS=100 bash scripts/smoke-crud.sh
```

## Integration tests

```bash
go test -tags integration ./tests/integration/admin_crud_test.go -v
```

Covers the full POST → GET → PUT → GET → DELETE lifecycle + read-only
enforcement + 404 mapping + filter correctness + pagination integrity.

## Production safety

In production environments:

- The compose file sets `EXCHANGEOS_ENABLE_ADMIN_API=true` for **local dev only**
- Production Helm values MUST set `--set api.adminApiEnabled=false`
- Even if enabled, the production Helm template ADDs `oauth2-proxy` in front of `/v1/admin/*` requiring the `exchangeos:admin` scope
- The `exchangeos_app` runtime DB role has SELECT/INSERT/UPDATE/DELETE but the admin operator should bind a **separate** DB role (`exchangeos_admin_ro` for inspection, `exchangeos_admin_rw` for mutations) gated by RBAC

See `docs/security/iso27001-controls-mapping.md` § 5.15 (access control) for the
full production hardening requirements before enabling.

## Cross-references

- [`scripts/smoke-crud.sh`](../../scripts/smoke-crud.sh) — count assertion smoke
- [`tests/integration/admin_crud_test.go`](../../tests/integration/admin_crud_test.go) — round-trip
- [`internal/adminapi/registry.go`](../../internal/adminapi/registry.go) — 30-table source of truth
- [`internal/adminapi/handler.go`](../../internal/adminapi/handler.go) — CRUD impl
- [`seeds/`](../../seeds/) — 14 SQL seed files
- [`migrations/`](../../migrations/) — 9 CRDB migrations

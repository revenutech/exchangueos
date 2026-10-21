-- ExchangeOS — 000001: tenants + actors + global setup
-- Idempotente. Roda via cmd/migrator.
-- DB: CockroachDB v24.3.32 via shared hub TLS (cockroachdb/modules/exchangeos/).

BEGIN;

-- NOTE: SET CLUSTER SETTING `sql.defaults.experimental_temporary_tables.enabled`
-- moved to compose/crdb-init (it can't live inside a transaction; CRDB rejects
-- SET CLUSTER SETTING inside BEGIN/COMMIT). Production hub provisioning does
-- the same — see cockroachdb/modules/exchangeos/users.sql.

-- ─────────────────────────────────────────────────────────────────────────────
-- tenants
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tenants (
    tenant_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code          STRING(64) NOT NULL UNIQUE,
    name          STRING(256) NOT NULL,
    country       STRING(2)  NOT NULL,
    status        STRING(32) NOT NULL DEFAULT 'ACTIVE',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    metadata      JSONB NOT NULL DEFAULT '{}'::JSONB,
    INDEX idx_tenants_status (status),
    INDEX idx_tenants_country (country)
);

-- ─────────────────────────────────────────────────────────────────────────────
-- actors (users + service accounts within tenants)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS actors (
    actor_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE RESTRICT,
    external_sub  STRING(256) NOT NULL,    -- OIDC sub (IdentityOS/Keycloak)
    type          STRING(32)  NOT NULL,    -- HUMAN | SERVICE
    display_name  STRING(256),
    status        STRING(32)  NOT NULL DEFAULT 'ACTIVE',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    UNIQUE (tenant_id, external_sub),
    INDEX idx_actors_tenant_status (tenant_id, status)
);

-- ─────────────────────────────────────────────────────────────────────────────
-- audit_events (envelope-of-envelopes)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_events (
    event_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(tenant_id),
    actor_id        UUID REFERENCES actors(actor_id),
    correlation_id  STRING(64) NOT NULL,
    causation_id    STRING(64),
    source          STRING(128) NOT NULL,
    event_type      STRING(128) NOT NULL,
    schema_version  STRING(16)  NOT NULL,
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    INDEX idx_audit_correlation (correlation_id),
    INDEX idx_audit_tenant_type_time (tenant_id, event_type, occurred_at DESC),
    INDEX idx_audit_source (source)
);

-- ─────────────────────────────────────────────────────────────────────────────
-- schema_migrations (golang-migrate compat)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS schema_migrations (
    version  BIGINT PRIMARY KEY,
    dirty    BOOL NOT NULL DEFAULT false
);

COMMIT;

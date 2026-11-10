-- ExchangeOS — 000010: archive_policy + archive_manifest (MS-024b 1.1, RTSK.024.1001.01)
--
-- Data-driven policy for cmd/archiver (docs/security/data-lifecycle/archival-cron.md):
-- one row per table that has an archival rule in docs/security/data-lifecycle/README.md.
-- The five tables whose archival is «Never» there (currencies, calendars, currency_pairs,
-- netting_cutoffs, bic_records) have no row: age_threshold is NOT NULL and they have none.
--
-- Platform configuration and audit of the archiver itself, not tenant data: neither table
-- carries tenant_id. The archived rows keep their own tenant_id inside the batch.
--
-- retention_after NULL + legal_hold TRUE = never hard-deleted from CRDB (regulatory hold,
-- «NEVER» in the README's right-to-erasure column). Erasable tables get the README retention.
-- The archiver acts only on rows already archived (archival-cron.md, step 7).

BEGIN;

CREATE TABLE IF NOT EXISTS archive_policy (
    table_name       STRING(63)  PRIMARY KEY,
    age_threshold    INTERVAL    NOT NULL,      -- archive rows older than this
    retention_after  INTERVAL,                  -- hard-delete from CRDB after this (NULL = keep)
    gcs_prefix       STRING(128) NOT NULL,
    legal_hold       BOOL        NOT NULL DEFAULT false,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),

    CONSTRAINT chk_archive_policy_age_positive       CHECK (age_threshold > INTERVAL '0'),
    CONSTRAINT chk_archive_policy_retention_positive CHECK (retention_after IS NULL OR retention_after > INTERVAL '0'),
    CONSTRAINT chk_archive_policy_hold_keeps         CHECK (NOT legal_hold OR retention_after IS NULL),
    CONSTRAINT chk_archive_policy_prefix             CHECK (gcs_prefix ~ '^[a-z0-9_]+/$')
);

-- The manifest is itself C3 and never deleted (audit chain, archival-cron.md).
CREATE TABLE IF NOT EXISTS archive_manifest (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name    STRING(63)  NOT NULL REFERENCES archive_policy(table_name) ON DELETE RESTRICT,
    batch_uuid    UUID        NOT NULL UNIQUE,
    row_count     INT         NOT NULL,
    sha256        STRING(64)  NOT NULL,
    gcs_uri       STRING(512) NOT NULL,
    completed_at  TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    legal_hold    BOOL        NOT NULL DEFAULT false,

    CONSTRAINT chk_archive_manifest_rows   CHECK (row_count > 0),
    CONSTRAINT chk_archive_manifest_sha256 CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_archive_manifest_uri    CHECK (gcs_uri LIKE 'gs://%'),

    INDEX idx_archive_manifest_table_completed (table_name, completed_at DESC)
);

-- Seed: 24 tables, mirroring the README retention schedule. ON CONFLICT DO NOTHING keeps the
-- migration idempotent and never overwrites a policy an operator changed afterwards.
INSERT INTO archive_policy (table_name, age_threshold, retention_after, gcs_prefix, legal_hold) VALUES
    -- regulatory hold (README: right-to-erasure NEVER)
    ('audit_events',              INTERVAL '1 year',   NULL,                'audit_events/',              true),
    ('fx_trades',                 INTERVAL '1 year',   NULL,                'fx_trades/',                 true),
    ('trade_amendments',          INTERVAL '1 year',   NULL,                'trade_amendments/',          true),
    ('cls_cycles',                INTERVAL '1 year',   NULL,                'cls_cycles/',                true),
    ('cls_cycle_trades',          INTERVAL '1 year',   NULL,                'cls_cycle_trades/',          true),
    ('payin_instructions',        INTERVAL '1 year',   NULL,                'payin_instructions/',        true),
    ('net_reports',               INTERVAL '1 year',   NULL,                'net_reports/',               true),
    ('classifications',           INTERVAL '1 year',   NULL,                'classifications/',           true),
    ('iof_computations',          INTERVAL '1 year',   NULL,                'iof_computations/',          true),
    ('bacen_reports',             INTERVAL '1 year',   NULL,                'bacen_reports/',             true),
    ('system_events',             INTERVAL '1 year',   NULL,                'system_events/',             true),
    ('positions',                 INTERVAL '1 year',   NULL,                'positions/',                 true),
    -- retention then hard-delete (README retention column)
    ('tenants',                   INTERVAL '1 year',   INTERVAL '7 years',  'tenants/',                   false),
    ('actors',                    INTERVAL '1 year',   INTERVAL '5 years',  'actors/',                    false),
    ('counterparties',            INTERVAL '1 year',   INTERVAL '7 years',  'counterparties/',            false),
    ('ssis',                      INTERVAL '1 year',   INTERVAL '7 years',  'ssis/',                      false),
    ('screening_results',         INTERVAL '1 year',   INTERVAL '7 years',  'screening_results/',         false),
    ('risk_limits',               INTERVAL '1 year',   INTERVAL '5 years',  'risk_limits/',               false),
    ('quotes',                    INTERVAL '90 days',  INTERVAL '5 years',  'quotes/',                    false),
    ('rfqs',                      INTERVAL '90 days',  INTERVAL '5 years',  'rfqs/',                      false),
    ('eod_jobs',                  INTERVAL '90 days',  INTERVAL '1 year',   'eod_jobs/',                  false),
    ('quote_streams',             INTERVAL '30 days',  INTERVAL '1 year',   'quote_streams/',             false),
    ('outbox_events',             INTERVAL '90 days',  INTERVAL '7 years',  'outbox_events/',             false),
    ('outbox_dispatched_archive', INTERVAL '90 days',  INTERVAL '7 years',  'outbox_dispatched_archive/', false)
ON CONFLICT (table_name) DO NOTHING;

COMMIT;

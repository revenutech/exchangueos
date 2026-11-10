-- ExchangeOS — 000011: cos_cases + cos_case_transitions (MS-024g 1.2, RTSK.024.1402.01)
--
-- Persistence of the COSCase aggregate (modules/compliance/domain/cos_case.go): a Comunicação de Operação Suspeita
-- on its way to SISCOAF. Lifecycle DRAFT → UNDER_REVIEW → APPROVED → SUBMITTED → ACCEPTED | REJECTED, with 4-eyes
-- (two distinct approvers, neither the creator). RN_FX_039: the COS goes to SISCOAF within 1 business day after the
-- decision; decided_at is the instant the deadline counts from.
--
-- The schema repeats the aggregate's invariants as CHECKs, so a row written around the domain cannot hold a state
-- the domain refuses. payload_hash is the SHA-256 (hex) of the document sent to SISCOAF, set at SUBMITTED.
--
-- PII (C4): narrative and rejection_reason describe a person. Secrecy of the COS (Lei 9.613/1998, art. 11): the
-- reported person must never learn of it; these rows never leave the compliance module.
--
-- cos_case_transitions is the audit trail: one row per transition, never updated. ON DELETE RESTRICT on purpose
-- (not CASCADE): an audit row must not disappear with its case. Its FK is (tenant_id, cos_case_id), so a transition
-- cannot point at another tenant's case.
--
-- The link to a screening result lives in ONE place: screening_results.cos_case_id (000014, MS-024g 1.3), with a
-- tenant-scoped FK. cos_cases carries no screening_result_id, so the two sides cannot diverge.
--
-- Tenant: every FK out of these tables is (tenant_id, <id>), so a case or a transition can only name rows of its own
-- tenant. counterparties gets UNIQUE (tenant_id, counterparty_id) as the target of the counterparty FK.

BEGIN;

ALTER TABLE counterparties ADD CONSTRAINT IF NOT EXISTS uq_counterparties_tenant UNIQUE (tenant_id, counterparty_id);

COMMIT;

BEGIN;

CREATE TABLE IF NOT EXISTS cos_cases (
    cos_case_id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    counterparty_id      UUID        NOT NULL,
    created_by           STRING(128) NOT NULL,              -- token sub, never a typed name
    narrative            STRING      NOT NULL,              -- PII
    status               STRING(16)  NOT NULL DEFAULT 'DRAFT',
    approvers            STRING[]    NOT NULL DEFAULT '{}', -- token subs, in approval order
    decided_at           TIMESTAMPTZ,                       -- second approval (RN_FX_039 deadline starts here)
    submitted_at         TIMESTAMPTZ,
    responded_at         TIMESTAMPTZ,
    protocol_id          STRING(64),
    payload_hash         STRING(64),                        -- sha256 hex of the document sent to SISCOAF
    rejection_reason     STRING(1024),                      -- PII: SISCOAF's text may quote the narrative
    version              INT         NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),

    CONSTRAINT fk_cos_cases_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE RESTRICT,
    CONSTRAINT fk_cos_cases_counterparty FOREIGN KEY (tenant_id, counterparty_id) REFERENCES counterparties(tenant_id, counterparty_id) ON DELETE RESTRICT,

    CONSTRAINT chk_cos_cases_status CHECK (status IN ('DRAFT','UNDER_REVIEW','APPROVED','SUBMITTED','ACCEPTED','REJECTED')),
    -- 4-eyes: at most two approvers, distinct, neither the creator
    CONSTRAINT chk_cos_cases_four_eyes CHECK (
        coalesce(array_length(approvers, 1), 0) <= 2
        AND NOT (created_by = ANY (approvers))
        AND (coalesce(array_length(approvers, 1), 0) < 2 OR approvers[1] <> approvers[2])
    ),
    -- from APPROVED on: two approvers and the decision instant
    CONSTRAINT chk_cos_cases_decided CHECK (
        status IN ('DRAFT','UNDER_REVIEW')
        OR (coalesce(array_length(approvers, 1), 0) = 2 AND decided_at IS NOT NULL)
    ),
    -- from SUBMITTED on: submission instant, SISCOAF protocol and the hash of what was sent
    CONSTRAINT chk_cos_cases_submitted CHECK (
        status IN ('DRAFT','UNDER_REVIEW','APPROVED')
        OR (submitted_at IS NOT NULL AND protocol_id IS NOT NULL AND payload_hash IS NOT NULL)
    ),
    CONSTRAINT chk_cos_cases_responded CHECK ((status IN ('ACCEPTED','REJECTED')) = (responded_at IS NOT NULL)),
    CONSTRAINT chk_cos_cases_rejection CHECK ((status = 'REJECTED') = (rejection_reason IS NOT NULL)),
    CONSTRAINT chk_cos_cases_payload_hash CHECK (payload_hash IS NULL OR payload_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_cos_cases_version CHECK (version >= 1),
    -- target of tenant-scoped FKs (cos_case_transitions here, screening_results in 000014): a child row can only
    -- point at a case of its own tenant
    CONSTRAINT uq_cos_cases_tenant_case UNIQUE (tenant_id, cos_case_id),

    INDEX idx_cos_cases_tenant_status (tenant_id, status, created_at DESC),
    INDEX idx_cos_cases_counterparty (tenant_id, counterparty_id, created_at DESC),
    -- RN_FX_039 watch: approved and not yet submitted
    INDEX idx_cos_cases_pending_submission (tenant_id, decided_at) WHERE status = 'APPROVED'
);

CREATE TABLE IF NOT EXISTS cos_case_transitions (
    transition_id  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    cos_case_id    UUID        NOT NULL,
    from_status    STRING(16),                -- NULL for the creation (→ DRAFT)
    to_status      STRING(16)  NOT NULL,
    actor          STRING(128) NOT NULL,      -- token sub
    case_version   INT         NOT NULL,      -- cos_cases.version after the transition
    occurred_at    TIMESTAMPTZ NOT NULL,
    recorded_at    TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),

    CONSTRAINT fk_cos_case_transitions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE RESTRICT,
    CONSTRAINT fk_cos_case_transitions_case FOREIGN KEY (tenant_id, cos_case_id) REFERENCES cos_cases(tenant_id, cos_case_id) ON DELETE RESTRICT,
    CONSTRAINT chk_cos_case_transitions_from CHECK (from_status IS NULL OR from_status IN ('DRAFT','UNDER_REVIEW','APPROVED','SUBMITTED','ACCEPTED','REJECTED')),
    CONSTRAINT chk_cos_case_transitions_to CHECK (to_status IN ('DRAFT','UNDER_REVIEW','APPROVED','SUBMITTED','ACCEPTED','REJECTED')),
    CONSTRAINT uq_cos_case_transitions_version UNIQUE (cos_case_id, case_version),

    INDEX idx_cos_case_transitions_case (tenant_id, cos_case_id, occurred_at)
);

COMMIT;

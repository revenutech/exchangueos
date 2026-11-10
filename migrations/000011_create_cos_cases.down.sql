-- ExchangeOS — 000011 rollback

BEGIN;

DROP TABLE IF EXISTS cos_case_transitions;
DROP TABLE IF EXISTS cos_cases;

COMMIT;

BEGIN;

DROP INDEX IF EXISTS counterparties@uq_counterparties_tenant CASCADE;

COMMIT;

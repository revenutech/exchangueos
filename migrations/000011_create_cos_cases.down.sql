-- ExchangeOS — 000011 rollback

BEGIN;

DROP TABLE IF EXISTS cos_case_transitions;
DROP TABLE IF EXISTS cos_cases;

COMMIT;

-- ExchangeOS — 000014 rollback

BEGIN;

DROP INDEX IF EXISTS screening_results@idx_screen_cos_case;
ALTER TABLE screening_results DROP CONSTRAINT IF EXISTS fk_screening_results_cos_case;

COMMIT;

BEGIN;

ALTER TABLE screening_results DROP COLUMN IF EXISTS cos_case_id;

COMMIT;

-- ExchangeOS — 000014: screening_results.cos_case_id (MS-024g 1.3, RTSK.024.1403.01)
--
-- Optional link from a screening result to the COS case it led to (000011). NULL = no case: most screening results
-- never become a COS. The FK is tenant-scoped, (tenant_id, cos_case_id) → cos_cases(tenant_id, cos_case_id), so a
-- screening result can only point at a case of its own tenant. ON DELETE RESTRICT: a COS case is audit material
-- (Lei 9.613 art. 11) and is never deleted from under the screening result that references it.
--
-- Idempotent: IF NOT EXISTS on the column, the constraint and the index.

BEGIN;

ALTER TABLE screening_results ADD COLUMN IF NOT EXISTS cos_case_id UUID;

COMMIT;

BEGIN;

ALTER TABLE screening_results ADD CONSTRAINT IF NOT EXISTS fk_screening_results_cos_case
    FOREIGN KEY (tenant_id, cos_case_id) REFERENCES cos_cases(tenant_id, cos_case_id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_screen_cos_case ON screening_results (tenant_id, cos_case_id)
    WHERE cos_case_id IS NOT NULL;

COMMIT;

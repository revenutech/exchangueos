-- ExchangeOS — 000013 rollback

BEGIN;

DROP INDEX IF EXISTS idx_quotes_requester;

ALTER TABLE quotes DROP CONSTRAINT IF EXISTS chk_quotes_distinct_parties;
ALTER TABLE quotes DROP CONSTRAINT IF EXISTS chk_quotes_bic_len;
ALTER TABLE quotes DROP CONSTRAINT IF EXISTS chk_quotes_side;

ALTER TABLE quotes DROP COLUMN IF EXISTS side;
ALTER TABLE quotes DROP COLUMN IF EXISTS provider_bic;
ALTER TABLE quotes DROP COLUMN IF EXISTS requester_bic;

COMMIT;

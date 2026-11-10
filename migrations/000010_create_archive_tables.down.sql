-- ExchangeOS — 000010 rollback

BEGIN;

DROP TABLE IF EXISTS archive_manifest;
DROP TABLE IF EXISTS archive_policy;

COMMIT;

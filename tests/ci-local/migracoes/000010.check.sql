# 000010 archive_policy + archive_manifest (MS-024b 1.1). Each query must return t.
# 24 policies: one per table with an archival rule in docs/security/data-lifecycle/README.md.
SELECT count(*) = 24 FROM archive_policy
# the five «Never» tables have no policy
SELECT count(*) = 0 FROM archive_policy WHERE table_name IN ('currencies','calendars','currency_pairs','netting_cutoffs','bic_records')
# every policy names a table that exists
SELECT count(*) = 0 FROM archive_policy p WHERE NOT EXISTS (SELECT 1 FROM information_schema.tables t WHERE t.table_schema = 'public' AND t.table_name = p.table_name)
# regulatory hold never hard-deletes
SELECT count(*) = 12 FROM archive_policy WHERE legal_hold AND retention_after IS NULL
SELECT count(*) = 0 FROM archive_policy WHERE legal_hold AND retention_after IS NOT NULL
# manifest refuses a bad sha256 and a non-gs uri (CHECK constraints present)
SELECT count(*) = 3 FROM information_schema.check_constraints WHERE constraint_name IN ('chk_archive_manifest_rows','chk_archive_manifest_sha256','chk_archive_manifest_uri')

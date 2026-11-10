# 000011 cos_cases + cos_case_transitions (MS-024g 1.2). Each query must return t.
SELECT count(*) = 2 FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('cos_cases','cos_case_transitions')
# tenant scoping: tenant_id NOT NULL with FK to tenants, on both tables
SELECT count(*) = 2 FROM information_schema.columns WHERE table_name IN ('cos_cases','cos_case_transitions') AND column_name = 'tenant_id' AND is_nullable = 'NO'
SELECT count(*) = 2 FROM information_schema.referential_constraints WHERE constraint_name IN ('fk_cos_cases_tenant','fk_cos_case_transitions_tenant')
# lifecycle, 4-eyes and payload_hash enforced by the schema
SELECT count(*) = 7 FROM information_schema.check_constraints WHERE constraint_name IN ('chk_cos_cases_status','chk_cos_cases_four_eyes','chk_cos_cases_decided','chk_cos_cases_submitted','chk_cos_cases_responded','chk_cos_cases_rejection','chk_cos_cases_payload_hash')
SELECT count(*) = 1 FROM information_schema.columns WHERE table_name = 'cos_cases' AND column_name = 'payload_hash'
# optimistic concurrency
SELECT count(*) = 1 FROM information_schema.columns WHERE table_name = 'cos_cases' AND column_name = 'version' AND is_nullable = 'NO'

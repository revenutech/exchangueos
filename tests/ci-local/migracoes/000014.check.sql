# 000014 screening_results.cos_case_id (MS-024g 1.3). Each query must return t.
# optional column: a screening result may exist without a COS case
SELECT count(*) = 1 FROM information_schema.columns WHERE table_name = 'screening_results' AND column_name = 'cos_case_id' AND is_nullable = 'YES'
# tenant-scoped FK to cos_cases (tenant_id, cos_case_id)
SELECT count(*) = 1 FROM information_schema.referential_constraints WHERE constraint_name = 'fk_screening_results_cos_case'

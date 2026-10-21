-- ExchangeOS seeds — 07: actors + audit_events
-- 5 actors (system + 3 humans + 1 service) attached to DEV tenant.
-- 5 audit_events covering trade booking, amendment, screening, EOD, login.
-- Idempotent via ON CONFLICT DO NOTHING.

BEGIN;

-- ─── 5 actors ──────────────────────────────────────────────────────────────
INSERT INTO actors (actor_id, tenant_id, external_sub, type, display_name, status) VALUES
    ('11111111-0000-5000-8000-000000000001', '00000000-0000-5000-8000-000000000001', 'sub:system',          'SERVICE', 'System Service',          'ACTIVE'),
    ('11111111-0000-5000-8000-000000000002', '00000000-0000-5000-8000-000000000001', 'sub:ops_alice',       'HUMAN',   'Alice (Trader)',          'ACTIVE'),
    ('11111111-0000-5000-8000-000000000003', '00000000-0000-5000-8000-000000000001', 'sub:ops_bob',         'HUMAN',   'Bob (Compliance)',        'ACTIVE'),
    ('11111111-0000-5000-8000-000000000004', '00000000-0000-5000-8000-000000000001', 'sub:ops_carol',       'HUMAN',   'Carol (Risk)',            'ACTIVE'),
    ('11111111-0000-5000-8000-000000000005', '00000000-0000-5000-8000-000000000001', 'sub:integration_test','SERVICE', 'Integration Test Bot',    'ACTIVE')
ON CONFLICT (tenant_id, external_sub) DO NOTHING;

-- ─── 5 audit_events ────────────────────────────────────────────────────────
INSERT INTO audit_events (event_id, tenant_id, actor_id, correlation_id, causation_id, source, event_type, schema_version, payload, occurred_at) VALUES
    ('22222222-0000-5000-8000-000000000001', '00000000-0000-5000-8000-000000000001',
     '11111111-0000-5000-8000-000000000002', 'corr-001', NULL,
     'api.trade', 'TRADE_BOOKED', 'v1',
     '{"trade_id":"33333333-0000-5000-8000-000000000001","pair":"EURUSD","amount":"1000000"}'::JSONB,
     '2026-05-25T10:00:00Z'),

    ('22222222-0000-5000-8000-000000000002', '00000000-0000-5000-8000-000000000001',
     '11111111-0000-5000-8000-000000000003', 'corr-002', 'corr-001',
     'api.amendment', 'AMENDMENT_PROPOSED', 'v1',
     '{"trade_id":"33333333-0000-5000-8000-000000000001","field":"value_date"}'::JSONB,
     '2026-05-25T10:05:00Z'),

    ('22222222-0000-5000-8000-000000000003', '00000000-0000-5000-8000-000000000001',
     '11111111-0000-5000-8000-000000000003', 'corr-003', NULL,
     'compliance.screening', 'COUNTERPARTY_SCREENED', 'v1',
     '{"bic":"DEUTDEFF","risk_level":"LOW","hits":0}'::JSONB,
     '2026-05-25T10:10:00Z'),

    ('22222222-0000-5000-8000-000000000004', '00000000-0000-5000-8000-000000000001',
     '11111111-0000-5000-8000-000000000001', 'corr-004', NULL,
     'admin.eod', 'EOD_STARTED', 'v1',
     '{"business_date":"2026-05-25","tenant":"DEV"}'::JSONB,
     '2026-05-25T22:00:00Z'),

    ('22222222-0000-5000-8000-000000000005', '00000000-0000-5000-8000-000000000001',
     '11111111-0000-5000-8000-000000000002', 'corr-005', NULL,
     'identityos.auth', 'ACTOR_LOGIN', 'v1',
     '{"method":"oidc","ip":"127.0.0.1"}'::JSONB,
     '2026-05-25T09:30:00Z')
ON CONFLICT (event_id) DO NOTHING;

COMMIT;

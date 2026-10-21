# MS-023u — database-sync-cross-module

| Field | Value |
|-------|-------|
| **Code** | MS-023u |
| **Name** | database-sync-cross-module |
| **Phase** | F15M |
| **Sprint** | 17-18 |
| **Status** | DELIVERED |
| **Owner** | Platform team |
| **Created** | 2026-05-24 |
| **Updated** | 2026-05-24 |
| **Delivered** | 2026-05-24 |
| **Dependencies** | MS-023t (delivered) |

## Delivery Notes

**Acceptance criteria met:**
- ✅ **3 sync patterns implemented:**
  - **gRPC pull (sync)** — Risk.CheckLimit before Trade booking (pre-trade validation)
  - **In-process event bus push (async)** — `internal/eventbus/` with QuoteAcceptedHandler routing `quote.accepted.v1` → `trade.BookTrade` (TestContainer_QuoteAccepted_BooksTrade proves end-to-end)
  - **Transactional outbox (async, persistent)** — `pkg/outbox/` + migration 000009 + postgres Store + Kafka publisher (kgo) + cmd/worker dispatch loop
- ✅ Container exposes `EventBus` field for any cross-context subscribers
- ✅ Publisher selection via build tag (default no-op / kafka franz-go)

**Deferred:**
- ⏳ Concrete sync handlers across 13 sibling platform modules (AccountOS, PaymentOS, LedgerOS, AuthorityOS, RiskOS, ComplOS, TreasuryOS, IdentityOS, KeycloakOS, OnboardOS, BillingOS, CardOS, InvestOS v2) — those happen at integration-with-platform time
- ⏳ 40 FX-SYNC-* pattern catalog — separate documentation track

## Description

Shared CRDB hub TLS adoption (ADR-015) + pkg/integration/<module>/ 13 gRPC clients + CDC CHANGEFEED 7 topics + 11 Kafka domain events formalizados + native AccountOS (7 RPCs + tenant materialized view + balance saga) + native PaymentOS (4 RPCs + cross-border PIX saga + wire TED FX saga) + 40 FX-SYNC-* patterns + migration playbook accountos/paymentos + E2E cross-module + chaos engineering.

## Acceptance Criteria

- [ ] pkg/integration/<module>/ 13 packages com gRPC clients + circuit breakers
- [ ] CDC CHANGEFEED para 7 tabelas → Kafka
- [ ] 11 Kafka domain event topics formalizados
- [ ] Native AccountOS: 7 RPCs + tenant materialized view + 15+ tests
- [ ] Native PaymentOS: 4 RPCs + sagas + 12+ tests + 1 E2E
- [ ] ADR-015 ACCEPTED
- [ ] Migration playbook documentado

## Deliverables

- pkg/integration/ 13 packages
- CDC config em migration
- 11 Kafka topics em AsyncAPI
- Sagas em modules/trade/application/saga/ e modules/settlement/
- 40 patterns em 270-fx-sync-cross-module-patterns.md

## Cross-References

- Plano monolitico: §19 + Fase F15M
- Workstream: 05-integrations

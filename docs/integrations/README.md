# ExchangeOS — Sibling Module Integration Contracts

Revenu Platform consists of 13+ sibling modules. ExchangeOS integrates with the
following 5 directly today, plus 3 future siblings (RiskOS / ComplOS / TreasuryOS)
documented as design intents. Each contract document under this folder describes:

- What ExchangeOS **produces** (events + sync RPCs)
- What ExchangeOS **consumes** (events + sync RPCs)
- Failure semantics + retry policy
- Open questions + integration risks

## Integration matrix

| Sibling | Produces → | Consumes ← | Sync RPCs (both ways) | Maturity |
|---------|-----------|-----------|----------------------|----------|
| [LedgerOS](ledgeros.md)       | trade.settled.v1 → posts journal entries | none | Read PostingResult for reconciliation | 🟡 Spec |
| [AccountOS](accountos.md)     | none (consumes only) | tenant.created.v1 / counterparty.created.v1 | Resolve tenant_id from API token | ✅ Wired (concept) |
| [PaymentOS](paymentos.md)     | settlement.payin_requested.v1 | payment.settled.v1 | PvP coordination via CLS | 🟡 Spec |
| [AuthorityOS](authorityos.md) | compliance.report_ready.v1 | regulator.policy_updated.v1 | SISCOAF + BACEN submission proxy | 🟡 Spec |
| [IdentityOS / KeycloakOS](identityos.md) | none | identity.actor_disabled.v1 | OIDC validation per request | ✅ Wired |
| [RiskOS](riskos.md)           | risk.breach.v1 + position.snapshot.v1 | group_risk.limit_pressure.v1 | none initially | ⏳ Design |
| [ComplOS](complos.md)         | compliance.cos_required.v1 | sanctions.list_updated.v1 + policy.bacen_code_added.v1 | SanctionsScreening + ComplianceQuery | ⏳ Design |
| [TreasuryOS](treasuryos.md)   | settlement.payment_required.v1 + cls.payin_required.v1 | liquidity.unavailable.v1 + nostro.balance_snapshot.v1 + hedge.proposal.v1 | LiquidityQuery + PositionQuery | ⏳ Design |

## Out of scope (initial release)

| Sibling | Why deferred |
|---------|-------------|
| OnboardOS | Tenant onboarding triggers AccountOS first; ExchangeOS picks up via tenant.created event |
| BillingOS | Per-trade billing; downstream |
| CardOS / InvestOS v2 | Distinct product lines |

## Common contract conventions

All integration events use the canonical naming `<context>.<action>.v<N>` (FX-EDA-004).

All sync RPCs include `TenantContext{tenant_id, actor_id, correlation_id, causation_id}` as the first field.

Failure semantics by default:
- **Async events:** at-least-once delivery via Kafka outbox; consumers MUST be idempotent.
- **Sync RPCs:** caller times out at 5s with exponential backoff (3 retries); after that, falls back to the cached value if available.

## Versioning

Each sibling document carries a "Compatible since" header noting the minimum
ExchangeOS release that supports the listed contract. Breaking changes ship as
new event versions (v2) alongside v1 for one full release cycle before v1 is
removed.

## Open cross-module questions

See `.base/plans/00-governance/open-questions.md` for the cross-module list,
or per-doc "Open Questions" sections in each integration file.

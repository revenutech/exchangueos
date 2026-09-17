---
Code: RFLW.024.001.01
Domain: 024 (ExchangeOS)
Module: trade
Version: 1.0.0
Status: DRAFT
Title: Book FX Spot Trade via CLS (EUR/USD)
Traceability:
  - UserStory: US-FX-001 "Trader books spot trade"
  - RN: [RN_FX_001, RN_FX_002, RN_FX_010, RN_FX_026]
  - ISO20022: [fxtr.014.001.05]
  - Ontology: [exos:Spot, exos:CLS, exos:hasStatus]
Predecessor: —
Successor: RFLW.024.002.01 (Settle FX Trade via CLS PvP)
---

# RFLW.024.001.01 — Book FX Spot Trade via CLS

## Description

Trader (or counterparty) submits a spot quote that, on acceptance, becomes a
booked FX trade routed to CLS for PvP settlement on T+2.

## Pre-conditions

1. Counterparty (buyer + seller) BICs are CLS members (`bic_records.cls_eligible = true` via reda lookup).
2. Currency pair is CLS-eligible (18 CCYs).
3. Tenant has a non-breached COUNTERPARTY limit for the buyer BIC.

## Actors / Participants

- **Trader (UI)** — submits via REST + WebSocket
- **exchangeos-api** — REST/gRPC service
- **exchangeos.PricingEngine** — quotes spot mid + half-spread
- **exchangeos.QuoteService** — persists Quote + RFQ
- **exchangeos.RiskService** — pre-trade limit check
- **exchangeos.TradeService** — books FXTrade
- **exchangeos.SettlementService** — attaches trade to today's CLS cycle
- **Kafka outbox** — emits trade.created.v1 → downstream
- **CLS Bank (CLSBUS33)** — receives fxtr.014.001.05 envelope (out of scope here)

## Sequence

```mermaid
sequenceDiagram
    autonumber
    participant T as Trader
    participant API as exchangeos-api
    participant Q as QuoteService
    participant P as PricingEngine
    participant R as RiskService
    participant TR as TradeService
    participant CS as SettlementService
    participant KB as Kafka outbox

    T->>API: POST /v1/quotes (EUR/USD, 1M EUR notional)
    API->>P: GetMidRate(EUR,USD)
    P-->>API: mid=1.0800, halfSpread=0.0002
    API->>Q: GetQuote(...)
    Q-->>API: Quote{bid:1.0798, ask:1.0802, ttl:10s}
    API-->>T: 200 Quote

    T->>API: POST /v1/quotes/:id/accept
    API->>Q: AcceptQuote(quoteID, actor)
    Q-->>API: Quote{accepted, version:2}
    Note over Q,KB: emits quote.accepted.v1 via outbox

    KB-->>TR: dispatch quote.accepted.v1
    TR->>R: CheckLimit(COUNTERPARTY, buyerBIC, exposure=1.08M USD)
    R-->>TR: allowed=true
    TR->>TR: domain.NewFXTrade(...)
    TR->>CS: AttachTrade(today's cycle, tradeID)
    CS-->>TR: cycle{status:OPEN, trade_count:N+1}
    Note over TR,KB: persists fx_trades row + outbox trade.created.v1

    KB-->>API: dispatch trade.created.v1 (smoke endpoint exposes it)
    API-->>T: notification (WS) — Trade SETTLED on T+2 after CLS cycle close
```

## Error Flow

```mermaid
flowchart TB
    Start([POST accept quote]) --> Expired{Quote<br/>expired?}
    Expired -- yes --> ErrExp[/410 GONE — ErrQuoteExpired/]
    Expired -- no --> Limit{Limit<br/>breached?}
    Limit -- yes --> ErrLimit[/429 RESOURCE_EXHAUSTED — ErrBreached + breached_limit_id/]
    Limit -- no --> Validate{Validate<br/>RN_FX_001<br/>RN_FX_026?}
    Validate -- fail --> ErrVal[/400 INVALID_ARGUMENT/]
    Validate -- pass --> SaveOK[(persist + outbox)]
    SaveOK --> Notify([WS notify trader])
```

## Business Rules Applied

| Code | Rule |
|------|------|
| RN_FX_001 | Currency pair must be valid + ACTIVE in refdata |
| RN_FX_002 | Spot default T+2 (USD/CAD T+1) — handled by pricing.Tenor.ValueDate |
| RN_FX_010 | PvP via CLS for the 18 eligible CCYs |
| RN_FX_026 | NEVER float64 for money/rate — decimal.Decimal throughout |

## Observability

- **OTel spans:** GetQuote → AcceptQuote → CheckLimit → BookTrade → AttachTrade
- **Metrics:** `quote.accepted.v1` counter, `trade.created.v1` counter, `risk.breach` counter
- **Logs:** correlation_id propagated through TenantContext

## Compliance Notes

- BACEN: trade is classified post-book by ComplianceService (Circ 3.690 code, IOF computation).
- COS: not required for clear screening (LOW); HIGH-risk would trigger SISCOAF submission per RN_FX_039.

## Related Patterns

- FX-DDD-* (Domain-Driven Design)
- FX-EDA-* (Event-Driven Architecture — outbox dispatch)
- FX-IAM-* (TenantContext extraction)
- FX-OTEL-* (span propagation)

> **Nota de frescor 2026-09-17 (auditoria).** A pré-condição "BICs são membros CLS via `bic_records.cls_eligible`" está incorreta: a tabela `bic_records` (`migrations/000005_create_refdata.up.sql:41-53`) NÃO tem a coluna `cls_eligible`. A elegibilidade CLS é da MOEDA/PAR (`currencies.cls_eligible` em 000005:13, `currency_pairs.cls_eligible` em 000004:22), não do BIC.

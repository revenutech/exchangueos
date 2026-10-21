# ERD — Trade Domain

**Source migrations:** `migrations/000001_create_tenants.up.sql` + `migrations/000002_create_fx_trades.up.sql`

**Ontology grounding:** `.base/aasc/ontology/core/trade.ttl`

## Entities

```mermaid
erDiagram
    TENANTS ||--o{ COUNTERPARTIES         : "owns"
    TENANTS ||--o{ ACTORS                  : "owns"
    TENANTS ||--o{ FX_TRADES               : "owns"
    TENANTS ||--o{ TRADE_AMENDMENTS        : "owns"
    TENANTS ||--o{ AUDIT_EVENTS            : "owns"

    COUNTERPARTIES ||--o{ FX_TRADES        : "buyer_counterparty_id"
    COUNTERPARTIES ||--o{ FX_TRADES        : "seller_counterparty_id"
    FX_TRADES      ||--o{ TRADE_AMENDMENTS : "trade_id"
    ACTORS         ||--o{ TRADE_AMENDMENTS : "proposer / approver"

    TENANTS {
        uuid    tenant_id PK
        string  code UK
        string  name
        string  country
        string  status
        jsonb   metadata
        timestamptz created_at
        timestamptz updated_at
    }

    COUNTERPARTIES {
        uuid    counterparty_id PK
        uuid    tenant_id FK
        string  bic UK
        string  lei
        string  name
        string  country
        bool    cls_member
        bool    cfets_member
        string  status
    }

    FX_TRADES {
        uuid    trade_id PK
        uuid    tenant_id FK
        string  external_ref
        string  trade_type           "SPOT | FORWARD | NDF | SWAP"
        string  status               "PENDING|CONFIRMED|SETTLING|SETTLED|CANCELLED|REJECTED"
        string  settlement_venue     "CLS | BILATERAL | CFETS"
        uuid    buyer_counterparty_id FK
        uuid    seller_counterparty_id FK
        string  bought_currency
        decimal bought_amount        "DECIMAL(36,18)"
        string  sold_currency
        decimal sold_amount          "DECIMAL(36,18)"
        decimal deal_rate            "DECIMAL(36,18)"
        timestamptz trade_date
        date    value_date
        timestamptz confirmed_at
        timestamptz settled_at
        string  iso20022_message_id
        uuid    cls_cycle_id
    }

    TRADE_AMENDMENTS {
        uuid    amendment_id PK
        uuid    trade_id FK
        uuid    tenant_id FK
        string  status               "PROPOSED|APPROVED|REJECTED|APPLIED"
        string  change_type          "RATE|AMOUNT|VALUE_DATE|COUNTERPARTY"
        jsonb   before_payload
        jsonb   after_payload
        uuid    proposer_actor_id FK
        uuid    approver_actor_id FK
        timestamptz proposed_at
        timestamptz approved_at
        timestamptz applied_at
    }

    ACTORS {
        uuid    actor_id PK
        uuid    tenant_id FK
        string  external_sub         "OIDC sub from IdentityOS/Keycloak"
        string  type                 "HUMAN | SERVICE"
        string  display_name
        string  status
    }

    AUDIT_EVENTS {
        uuid    event_id PK
        uuid    tenant_id FK
        uuid    actor_id FK
        string  correlation_id
        string  causation_id
        string  source
        string  event_type
        string  schema_version
        jsonb   payload
        timestamptz occurred_at
        timestamptz recorded_at
    }
```

## Constraints

- `FX_TRADES.bought_currency <> sold_currency` (RN_FX_001)
- `FX_TRADES.{bought,sold}_amount > 0` (RN_FX_026)
- `FX_TRADES.deal_rate > 0`
- `FX_TRADES.status` ∈ enum (CHECK)
- `FX_TRADES.trade_type` ∈ {SPOT,FORWARD,NDF,SWAP}
- `FX_TRADES.settlement_venue` ∈ {CLS,BILATERAL,CFETS}
- `TRADE_AMENDMENTS.status` ∈ {PROPOSED,APPROVED,REJECTED,APPLIED}

## Indexes

- `idx_trades_tenant_status_value (tenant_id, status, value_date)` — main filter for List
- `idx_trades_venue_cycle (settlement_venue, cls_cycle_id) STORING (status)` — CLS scheduler
- `idx_trades_buyer / idx_trades_seller` — counterparty exposure aggregation
- `idx_trades_value_date WHERE status IN (CONFIRMED,SETTLING)` — partial settlement queue
- `idx_trades_iso_msg WHERE iso20022_message_id IS NOT NULL` — inbound message correlation

## Related ERDs

- erd-quote-domain.md (RFQ + Quote)
- erd-settlement-domain.md (cls_cycles + payin + net_reports)
- erd-risk-position-domain.md (limits + positions)
- erd-compliance-admin-domain.md

# Architecture Overview — ExchangeOS

> Cached snapshot. Refresh manually quando architecture muda significativamente.

## High-Level

```
┌────────────────────────────────────────────────────────────────┐
│  External: Traders / Counterparties / Services                 │
└────────────────────────────────────────────────────────────────┘
                            │ HTTPS + Bearer JWT (RS256) | mTLS
                            ▼
┌────────────────────────────────────────────────────────────────┐
│  KrakenD API Gateway :8080 — JWT validation + rate limit       │
└────────────────────────────────────────────────────────────────┘
                            │ gRPC
                            ▼
┌────────────────────────────────────────────────────────────────┐
│  ExchangeOS :8084 HTTP / :9084 gRPC                            │
│  ┌──────┬──────┬──────────┬───────┬─────────┬───────┐         │
│  │trade │quote │amendment │cls    │payin    │netrep │ ...     │
│  │      │      │          │settle │         │       │         │
│  └──────┴──────┴──────────┴───────┴─────────┴───────┘         │
└────────────────────────────────────────────────────────────────┘
                            │
        ┌───────┬───────┬───┴───┬──────┬────────┬────────┐
        ▼       ▼       ▼       ▼      ▼        ▼        ▼
   ┌──────┐ ┌────┐ ┌────┐ ┌─────┐ ┌─────┐ ┌──────┐ ┌──────┐
   │CRDB  │ │Kfk │ │OTel│ │Vault│ │ldgr │ │acct  │ │auth  │
   │hub   │ │    │ │    │ │     │ │ OS  │ │ OS   │ │OS    │
   │TLS   │ │    │ │    │ │     │ │     │ │      │ │      │
   └──────┘ └────┘ └────┘ └─────┘ └─────┘ └──────┘ └──────┘
```

## 14 Bounded Contexts

1. **trade** — FXTrade aggregate
2. **quote** — FXQuoteRequest (RFQ)
3. **amendment** — Amendment + Cancellation + Novation
4. **cls_settlement** — CLS PvP (fxtr 008-030)
5. **payin** — CLS PayIn cycle (camt.061/062/063)
6. **netreport** — CLS NetReport (camt.088)
7. **cfets_capture** — CFETS Trade Capture (fxtr 031-033)
8. **cfets_confirmation** — CFETS Confirmation (fxtr 034-038)
9. **settlement** — Non-CLS settlement (gross + BACEN Tx 70)
10. **refdata** — CurrencyPair, Calendar, SSI, Counterparty
11. **admin** — admi messages CLS (002/004/009/010/011/017)
12. **risk** — TradingLimit, NOP, VaR
13. **position** — Position keeping, MTM, P&L
14. **compliance** — DEC, SCE-*, IOF, COS, sanctions

## Sync Patterns

| Pattern | Use case |
|---------|----------|
| gRPC pull (sync) | Pre-trade validations (sanctions, credit, refdata) |
| CDC push (async) | Cross-module reactivity (CRDB CHANGEFEED → Kafka) |
| Kafka events (async) | Domain events + sagas (Outbox pattern) |

## Stack

| Layer | Tech |
|-------|------|
| Language | Go 1.25 |
| Database | CockroachDB v24.3.32 (shared hub TLS) |
| Messaging | Kafka KRaft 3-broker + IBM MQ bridge |
| Streaming | Apache Flink (NOP realtime + CEP fraud) |
| Observability | OpenTelemetry + Tempo + Mimir + Loki + Grafana |
| Auth | IdentityOS (gRPC :9084) + KeycloakOS v26.5.3 + Vault |
| API | gRPC + REST (OpenAPI 3.1) + AsyncAPI 3.0 |
| Container | Distroless multi-arch (amd64 + arm64) |
| Orchestration | GKE Autopilot 1.29+ + Helm + Argo Rollouts + GitOps ArgoCD |
| IaC | Terraform 1.5+ + GCP (KMS CMEK + WIF + VPC SC + Binary Auth) |
| CI/CD | GitHub Actions + SLSA L3 + Cosign + SBOM |
| Build | Task (taskfile.dev) + Makefile auto-gen + cross-platform |

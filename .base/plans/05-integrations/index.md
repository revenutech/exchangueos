# 05 — Integrations

> **Workstream:** Integrations
> **Versao:** 1.0.0
> **Status:** DRAFT

## Conteudo

| Documento | Status | Descricao |
|-----------|--------|-----------|
| `cls-orchestrator.md` | TODO | CLS Bank protocol integration (fxtr + PayIn + NetReport + admi) |
| `cfets-orchestrator.md` | TODO | CFETS PTPP integration (fxtr 031-038) |
| `bacen-orchestrator.md` | TODO | BACEN integration (Sistema Cambio + SISBACEN + SCE-IED + SCE-Credito + SCE-CBE + SISCOAF + SISCOMEX) |
| `swift-mt-bridge.md` | TODO | Legacy SWIFT MT bridge (MT300/MT304/MT202/MT202COV) via IBM MQ |
| `accountos-orchestrator.md` | TODO | AccountOS native integration (7 RPCs + tenant SoT + multi-CCY balance + CNR) |
| `paymentos-orchestrator.md` | TODO | PaymentOS native integration (4 RPCs + cross-border PIX + wire TED FX) |
| `ledgeros-orchestrator.md` | TODO | LedgerOS integration (PostMultiLegTransaction multi-CCY PvP) |
| `authorityos-orchestrator.md` | TODO | AuthorityOS integration (DEC + SCE-* + COS + BACEN Tx 70) |
| `riskos-orchestrator.md` | TODO | RiskOS integration (credit limit + NOP + VaR) |
| `complos-orchestrator.md` | TODO | ComplOS integration (sanctions + AML) |
| `treasuryos-orchestrator.md` | TODO | TreasuryOS integration (nostro + exposure) |
| `identityos-orchestrator.md` | TODO | IdentityOS integration (AuthZ Policy + 9 RPCs) |
| `keycloakos-orchestrator.md` | TODO | KeycloakOS integration (realm + 14 clients + Vault SPI) |
| `onboardos-integration.md` | TODO | OnboardOS Kafka events subscribe (kyc.completed) |
| `billingos-integration.md` | TODO | BillingOS Kafka events publish (trade.confirmed) |
| `database-sync-pattern.md` | TODO | Database Sync Pattern (gRPC pull + CDC push + Kafka events) — §19 monolitico |
| `pix-internacional-roadmap.md` | TODO | PIX Internacional Project Nexus (BIS Innovation Hub) — roadmap v2 |

## Integration Matrix (13 modulos)

Ver [`00-governance/integration-audit.md`](../00-governance/integration-audit.md) para matrix completa 4 vetores × 13 modulos.

## Sources

- §2.4 (SWIFT MT Bridge) + §19 (Database Sync Pattern + Native Cross-Module) do [snapshot monolitico v3.11.7](../_archive/allenty-v3.11.7-monolithic-plan.md)
- Pattern de referencia: [LedgerOS 05-integrations](../../../../ledgeros/.base/plans/05-integrations/)

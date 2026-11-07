# MS-024f — BACEN Submission Adapters (DEC + SCE-IED + SCE-Credito + SCE-CBE)

| Field | Value |
|-------|-------|
| **Code** | MS-024f |
| **Name** | bacen-submission-adapters |
| **Phase** | F-OPS-PROD |
| **Sprint** | 3 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | Compliance + Platform |
| **Dependencies** | MS-024e (nature codes), MS-024c (creds for BACEN gateway), MS-024h (postgres repos) |

## Why this milestone

Compliance domain classes for DEC + SCE-IED + SCE-Credito + SCE-CBE exist in `modules/compliance/domain/`. **Actual submission** to the BACEN gateway (RDR-S / SCE / DEC web service) is not implemented — `bacen_reports.status` only ever progresses to PENDING in tests. Real-world ExchangeOS needs to ship payloads to BACEN within regulatory deadlines.

## Description

Build 4 outbound adapters under `pkg/bacen/submission/` (one per filing type) implementing the BACEN gateway protocol (mTLS + signed XML + dated submission window). Wire into `modules/compliance/application` so `SubmitBACENReport` transitions PENDING → SUBMITTED → ACCEPTED/REJECTED via real API calls.

## Acceptance Criteria

- [ ] `pkg/bacen/submission/dec.go` — DEC (Declaração Eletrônica de Câmbio) submitter (Lei 14.286/2021, > USD 10K threshold)
- [ ] `pkg/bacen/submission/sce_ied.go` — SCE-IED (Investimento Estrangeiro Direto) submitter — 30 day registration window
- [ ] `pkg/bacen/submission/sce_credito.go` — SCE-Crédito (sucessor RDE-ROF, external credit) submitter
- [ ] `pkg/bacen/submission/sce_cbe.go` — SCE-CBE (Capitais Brasileiros no Exterior) submitter
- [ ] Common interface `Submitter.Submit(ctx, payload) (protocolID string, err error)` + `Query(ctx, protocolID) (Status, err)`
- [ ] mTLS client config via Vault-sourced cert
- [ ] XML payload generated from existing `BACENReport` aggregate (no hand-crafted XML in adapters — round-trip through `pkg/iso20022` style marshaller)
- [ ] XML signature (XAdES-BES) per BACEN technical spec
- [ ] Retry with exponential backoff + circuit breaker (5xx for > 60s → open)
- [ ] Mock BACEN gateway in `tests/integration/bacen_mock_server.go` for CI
- [ ] Integration test: submit one of each report type → poll status → assert ACCEPTED
- [ ] Helm sidecar / network policy for BACEN gateway egress (whitelisted IP)
- [ ] Compliance `SubmitBACENReport` use-case wired to real submitter via DI
- [ ] Metrics: `bacen_submission_total{type,status}` + `bacen_submission_duration_seconds{type}` + `bacen_submission_failed_total{type,reason}`
- [ ] Runbook covering common rejections + how to amend + resubmit

## Deliverables

- `pkg/bacen/submission/{dec,sce_ied,sce_credito,sce_cbe}.go`
- `pkg/bacen/submission/client.go` (mTLS + XAdES)
- `pkg/bacen/submission/marshal.go` (aggregate → XML)
- `tests/integration/bacen_mock_server.go`
- `tests/integration/bacen_submission_test.go`
- `deploy/k8s/network-policies/bacen-egress.yaml`
- `docs/operations/bacen-runbook.md`

## Cross-References

- Lei 14.286/2021 + Res BCB 277-561
- `modules/compliance/domain/bacen_report.go`
- `pkg/iso20022/` marshaller pattern
- ISO 27001 control 5.31 (legal/statutory/regulatory)
- MS-024e (nature codes feed into DEC)
- MS-024g (SISCOAF parallel filing)

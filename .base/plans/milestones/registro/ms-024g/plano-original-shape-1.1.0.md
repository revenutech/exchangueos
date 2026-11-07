# MS-024g — SISCOAF COS Submission

| Field | Value |
|-------|-------|
| **Code** | MS-024g |
| **Name** | siscoaf-cos-submission |
| **Phase** | F-OPS-PROD |
| **Sprint** | 3 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | Compliance + Platform |
| **Dependencies** | MS-024d (live sanctions surface hits), MS-024c (creds) |

## Why this milestone

`modules/compliance/domain/screening.go` produces `ScreeningResult.RequiresCOS=true` for HIGH-risk hits (RN_FX_039), but no code submits the COS (Comunicação de Operação Suspeita) to SISCOAF. Regulatory deadline is **1 business day** from decision — manual filing won't scale.

## Description

Implement SISCOAF COS XML generation + submission adapter. Includes case management workflow (4-eyes review queue + Compliance Officer approval before transmission), structured XML per COAF spec, mTLS submission, audit trail, and retry/resubmit.

## Acceptance Criteria

- [ ] `modules/compliance/domain/cos_case.go` — COSCase aggregate (DRAFT → UNDER_REVIEW → APPROVED → SUBMITTED → ACCEPTED/REJECTED) + 4-eyes approver tracking
- [ ] `pkg/siscoaf/cos.go` — COS XML marshaller per COAF technical specification (2026 schema version)
- [ ] `pkg/siscoaf/submitter.go` — mTLS client + SOAP/REST submission per current SISCOAF API
- [ ] Migration extending `screening_results` to optionally reference a `cos_cases.id`
- [ ] Migration creating `cos_cases` table with full lifecycle + audit + payload_hash
- [ ] Application service `OpenCOSCase` triggered automatically on HIGH-risk screening hit
- [ ] Review queue API: `GET /v1/compliance/cos-cases?status=UNDER_REVIEW` + `POST /v1/compliance/cos-cases/:id/approve` (requires `compliance_officer` role)
- [ ] Auto-submit on second approver sign-off
- [ ] SLA monitoring: alert if any case > 20h in UNDER_REVIEW (RN_FX_039 = 1 business day total)
- [ ] Integration test: open case → 4-eyes approve → submit to mock SISCOAF → assert ACCEPTED
- [ ] Metrics: `cos_cases_total{status}`, `cos_sla_breaches_total`, `cos_submission_duration_seconds`
- [ ] Runbook covering: rejected submission, resubmit, regulatory query response

## Deliverables

- `modules/compliance/domain/cos_case.go` + tests
- `modules/compliance/application/cos_workflow.go`
- `modules/compliance/api/cos_grpc.go` (under `grpcgen` tag) + REST handler
- `pkg/siscoaf/cos.go` + `submitter.go`
- `migrations/000011_create_cos_cases.up.sql` + `.down.sql`
- `tests/integration/cos_workflow_test.go`
- `docs/compliance/cos-workflow.md`

## Cross-References

- COAF resolução / SISCOAF API spec
- `modules/compliance/domain/screening.go` (RN_FX_039)
- `docs/security/sod-matrix.md` — 4-eyes for COS approval
- ISO 27001 control 5.31
- MS-024d (sanctions hits drive COS creation)

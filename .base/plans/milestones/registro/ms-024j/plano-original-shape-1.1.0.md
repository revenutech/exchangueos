# MS-024j — E2E Scenario Completion (7 remaining)

| Field | Value |
|-------|-------|
| **Code** | MS-024j |
| **Name** | e2e-scenario-completion |
| **Phase** | F-OPS-PROD |
| **Sprint** | 2 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | QA + All BC owners |
| **Dependencies** | MS-024h (postgres repos), MS-024d (sanctions surface) |

## Why this milestone

`tests/e2e/README.md` catalogues 10 canonical scenarios. Three implemented to date (01 EUR/USD spot, 05a/b risk-breach 404+400, 08 BACEN /version sanity). Seven remain — and they cover the most regulatory-and-customer-visible flows.

## Description

Implement the remaining 7 E2E scenarios against the full `make local-up` stack (CRDB + Kafka + Vault + OTel collector + api + worker). Use the existing harness (`tests/e2e/harness.go`) with `require.Eventually` (never `time.Sleep`). Each scenario corresponds to a real-world business journey worth a stakeholder demo.

## Acceptance Criteria

- [ ] **Scenario 02** — USD/BRL NDF: create RFQ → quote → accept → trade booked with NDF terms → fixing-date fetch → settlement amount computation → settled
- [ ] **Scenario 03** — CFETS capture: counterparty submits capture → fxtr.031 → ack fxtr.032 → notification fxtr.033 → CFETS deal ID returned
- [ ] **Scenario 04** — CFETS confirmation: 2 sides submit → pairing logic → fxtr.034/035 → CONFIRMED
- [ ] **Scenario 06** — Position update: book 3 trades same CCY pair → assert net position = sum legs → MTM with fresh fixing
- [ ] **Scenario 07** — CLS daily cycle: open cycle → attach trades → PayIn window → SETTLING → CLOSED with NetReport
- [ ] **Scenario 09** — Sanctions screening: counterparty matches OFAC fixture → ScreeningResult HIGH → COS case opened
- [ ] **Scenario 10** — EOD batch: trigger EOD → PTAX fetch → MTM → position snapshot → BACEN report ready → assert outbox events emitted in order
- [ ] Each scenario tagged `//go:build e2e` + uses `harness.Eventually` with explicit deadlines
- [ ] CI workflow `e2e.yml` boots `docker compose up -d`, runs all 10, captures container logs on failure
- [ ] Coverage report shows ≥ 60% of API surface exercised by E2E flow
- [ ] Each scenario writeup in `tests/e2e/scenarios/NN-<name>.md` (business context + steps + acceptance)

## Deliverables

- `tests/e2e/scenario_02_ndf_test.go`
- `tests/e2e/scenario_03_cfets_capture_test.go`
- `tests/e2e/scenario_04_cfets_confirmation_test.go`
- `tests/e2e/scenario_06_position_test.go`
- `tests/e2e/scenario_07_cls_cycle_test.go`
- `tests/e2e/scenario_09_sanctions_test.go`
- `tests/e2e/scenario_10_eod_test.go`
- `tests/e2e/scenarios/0N-<name>.md` × 7
- `.github/workflows/e2e.yml`
- Test fixtures `tests/e2e/fixtures/`

## Cross-References

- `tests/e2e/README.md` — catalogue + harness
- `tests/e2e/harness.go` — Eventually + GET helpers
- MS-023g (delivered harness + 3 of 10)
- MS-024d (sanctions for scenario 09)
- MS-024f (BACEN submission referenced in 10)

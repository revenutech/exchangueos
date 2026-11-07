# MS-024j — E2E Scenario Completion (7 restantes)

**Code:** `RMLS.024.016.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 do CONVENTIONS 1.7.0, ainda não em `main`): M=16; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** QA + All BC owners
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024j/plano-original-shape-1.1.0.md](../registro/ms-024j/plano-original-shape-1.1.0.md)

## Por quê

`tests/e2e/README.md` cataloga 10 cenários canônicos. Três implementados (01 EUR/USD spot, 05a/b risk-breach 404+400, 08 BACEN /version sanity). Sete restam — e cobrem os fluxos mais visíveis a regulador e cliente.

## Descrição

Implementar os 7 cenários E2E remanescentes contra o stack `make local-up` completo (CRDB + Kafka + Vault + OTel collector + api + worker). Usar o harness existente (`tests/e2e/harness.go`) com `require.Eventually` (nunca `time.Sleep`). Cada cenário corresponde a uma jornada de negócio real que serve de demo a stakeholder.

## Fase 1 — Implementação

### Etapa 1.1 — Cenários Spot/NDF/CFETS

- **1.1** `RTSK.024.1601.01` ✏️ Cenário 02 — USD/BRL NDF end-to-end. *Aceite:* RFQ → quote → accept → trade booked → fixing-date fetch → settlement amount → settled; `require.Eventually` com deadline explícito. *Entrega:* `tests/e2e/scenario_02_ndf_test.go` + writeup. *Escopo:* `tests/e2e/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.2** `RTSK.024.1602.01` ✏️ Cenário 03 — CFETS capture. *Aceite:* counterparty submete capture → fxtr.031 → ack fxtr.032 → notification fxtr.033 → CFETS deal ID retornado. *Entrega:* `tests/e2e/scenario_03_cfets_capture_test.go` + writeup. *Escopo:* `tests/e2e/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.3** `RTSK.024.1603.01` ✏️ Cenário 04 — CFETS confirmation. *Aceite:* 2 lados submetem → pairing → fxtr.034/035 → CONFIRMED. *Entrega:* `tests/e2e/scenario_04_cfets_confirmation_test.go` + writeup. *Escopo:* `tests/e2e/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Posição + CLS

- **1.4** `RTSK.024.1604.01` ✏️ Cenário 06 — update de posição. *Aceite:* 3 trades mesmo par → posição = soma legs → MTM com fixing fresco. *Entrega:* `tests/e2e/scenario_06_position_test.go` + writeup. *Escopo:* `tests/e2e/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.5** `RTSK.024.1605.01` ✏️ Cenário 07 — ciclo CLS diário. *Aceite:* open cycle → attach trades → PayIn window → SETTLING → CLOSED com NetReport. *Entrega:* `tests/e2e/scenario_07_cls_cycle_test.go` + writeup. *Escopo:* `tests/e2e/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Sanctions + EOD

- **1.6** `RTSK.024.1606.01` ✏️ Cenário 09 — sanctions screening. *Aceite:* counterparty match OFAC fixture → ScreeningResult HIGH → caso COS aberto. *Entrega:* `tests/e2e/scenario_09_sanctions_test.go` + writeup. *Escopo:* `tests/e2e/`. *Depende:* MS-024d (sanctions providers reais) — cross-MS. *Espera:* sessão dona — implementação (após MS-024d). [registro](../registro/README.md)
- **1.7** `RTSK.024.1607.01` ✏️ Cenário 10 — batch EOD. *Aceite:* trigger EOD → PTAX fetch → MTM → position snapshot → BACEN report ready → eventos de outbox na ordem. *Entrega:* `tests/e2e/scenario_10_eod_test.go` + writeup. *Escopo:* `tests/e2e/`. *Depende:* MS-024f (BACEN submission) — cross-MS. *Espera:* sessão dona — implementação (após MS-024f). [registro](../registro/README.md)

### Etapa 1.4 — CI + cobertura

- **1.8** `RTSK.024.1608.01` ✏️ Workflow `e2e.yml` boota `docker compose up -d`, roda 10 cenários, captura logs em falha. *Aceite:* workflow verde em branch de PR com 10 cenários. *Entrega:* `.github/workflows/e2e.yml`. *Escopo:* `.github/workflows/`. *Depende:* 1.1-1.7. *Espera:* Baremetal — runner CI com recurso. [registro](../registro/README.md)
- **1.9** `RTSK.024.1609.01` ✏️ Cobertura ≥ 60% da superfície de API exercida por E2E. *Aceite:* relatório gerado pelo CI mostra ≥ 60%. *Entrega:* configuração no workflow. *Escopo:* `.github/workflows/`. *Depende:* 1.8. *Espera:* gate — ratchet de coverage. [registro](../registro/README.md)
- **1.10** `RTSK.024.1610.01` ✏️ Fixtures E2E completas. *Aceite:* fixtures para 7 cenários novos existem em `tests/e2e/fixtures/`. *Entrega:* arquivos fixture. *Escopo:* `tests/e2e/fixtures/`. *Depende:* 1.1-1.7. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1611.01` ✏️ Rodar aceite da MS. *Aceite:* todas as tarefas da Fase 1 cumpridas + 3 execuções consecutivas do workflow verdes sem flake. *Entrega:* relatório em `registro/ms-024j/`. *Escopo:* módulo inteiro. *Espera:* gate — rodada N2 + e2e. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1612.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `tests/e2e/README.md` — catálogo + harness
- `tests/e2e/harness.go` — Eventually + GET helpers
- MS-023g (entregou harness + 3 de 10)
- MS-024d (sanctions para cenário 09)
- MS-024f (BACEN submission para cenário 10)

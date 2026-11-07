# MS-024g — SISCOAF COS Submission

**Code:** `RMLS.024.???.01` ⚠️ FFF aguarda decisão do dono em § 3.6 do CONVENTIONS 1.6.0 (DDD 024 auto-alocado fora do registro).
**Status:** BACKLOG
**Owner:** Compliance + Platform
**Fonte original:** `.base/plans/milestones/backlog/MS-024g-siscoaf-cos-submission.md` (shape 1.1.0, substituída por este arquivo)

## Por quê

`modules/compliance/domain/screening.go` produz `ScreeningResult.RequiresCOS=true` para hits HIGH (RN_FX_039), mas nenhum código submete o COS (Comunicação de Operação Suspeita) ao SISCOAF. Prazo regulatório é **1 dia útil** da decisão — filing manual não escala.

## Descrição

Implementar geração + submissão de XML COS para SISCOAF. Inclui workflow de gestão de caso (fila de revisão 4-eyes + aprovação do Compliance Officer antes de transmitir), XML estruturado por spec COAF, submissão mTLS, trilha de auditoria e retry/resubmit.

## Fase 1 — Implementação

### Etapa 1.1 — Aggregate + migração

- **1.1** `RTSK.024.???.01` ✏️ Aggregate `COSCase` (DRAFT → UNDER_REVIEW → APPROVED → SUBMITTED → ACCEPTED/REJECTED) + tracking 4-eyes. *Aceite:* transições válidas aceitas, inválidas recusadas, 1 aprovador não submete. *Entrega:* `modules/compliance/domain/cos_case.go` + testes. *Escopo:* `modules/compliance/domain/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.2** `RTSK.024.???.01` ✏️ Migração criando `cos_cases` com lifecycle completo + audit + `payload_hash`. *Aceite:* migração up+down idempotente, FK para tenant. *Entrega:* `migrations/000011_create_cos_cases.up.sql` + `.down.sql`. *Escopo:* `migrations/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.3** `RTSK.024.???.01` ✏️ Migração estendendo `screening_results` com FK opcional para `cos_cases.id`. *Aceite:* FK não obrigatória, migração up+down. *Entrega:* migração em `migrations/`. *Escopo:* `migrations/`. *Depende:* 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Workflow + wiring

- **1.4** `RTSK.024.???.01` ✏️ Application service `OpenCOSCase` acionado automático em hit HIGH. *Aceite:* `ScreenCounterparty` HIGH cria `COSCase` em DRAFT com payload_hash. *Entrega:* `modules/compliance/application/cos_workflow.go`. *Escopo:* `modules/compliance/application/`. *Depende:* 1.1. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.5** `RTSK.024.???.01` ✏️ Auto-submit após sign-off do segundo aprovador. *Aceite:* 2ª aprovação transiciona para APPROVED e dispara submissão. *Entrega:* lógica em `cos_workflow.go`. *Escopo:* `modules/compliance/application/`. *Depende:* 1.4. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Fila de revisão + API

- **1.6** `RTSK.024.???.01` ✏️ Endpoint `GET /v1/compliance/cos-cases?status=UNDER_REVIEW`. *Aceite:* retorna fila paginada com filter por status + tenant. *Entrega:* REST handler + gRPC handler sob `grpcgen`. *Escopo:* `modules/compliance/api/`. *Depende:* 1.4. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.7** `RTSK.024.???.01` ✏️ Endpoint `POST /v1/compliance/cos-cases/:id/approve` (role `compliance_officer`). *Aceite:* 403 sem role; sucesso grava aprovador + timestamp. *Entrega:* handler em `modules/compliance/api/cos_grpc.go`. *Escopo:* `modules/compliance/api/`. *Depende:* 1.4. *Espera:* dono — definição do papel `compliance_officer` no IdentityOS. [registro](../registro/README.md)

### Etapa 1.4 — Submissão + marshal

- **1.8** `RTSK.024.???.01` ✏️ Marshal COS XML por spec COAF (2026 schema). *Aceite:* round-trip valida contra XSD COAF. *Entrega:* `pkg/siscoaf/cos.go`. *Escopo:* `pkg/siscoaf/`. *Espera:* dono — confirmação da versão de schema. [registro](../registro/README.md)
- **1.9** `RTSK.024.???.01` ✏️ Cliente mTLS + SOAP/REST per API SISCOAF. *Aceite:* conexão contra mock devolve protocolID; retry em 5xx. *Entrega:* `pkg/siscoaf/submitter.go`. *Escopo:* `pkg/siscoaf/`. *Depende:* 1.8. *Espera:* dono — provisionamento da cert SISCOAF. [registro](../registro/README.md)

### Etapa 1.5 — SLA + observabilidade

- **1.10** `RTSK.024.???.01` ✏️ Alerta SLA: caso > 20h em UNDER_REVIEW (RN_FX_039 = 1 dia útil total). *Aceite:* alerta dispara em teste de injeção. *Entrega:* regra em `deploy/helm/` + métrica. *Escopo:* `deploy/helm/`, `modules/compliance/`. *Depende:* 1.4. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.11** `RTSK.024.???.01` ✏️ Métricas `cos_cases_total{status}`, `cos_sla_breaches_total`, `cos_submission_duration_seconds`. *Aceite:* 3 métricas expostas. *Entrega:* instrumentação em `cos_workflow.go`. *Escopo:* `modules/compliance/`. *Depende:* 1.5, 1.9. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.6 — Runbook + teste

- **1.12** `RTSK.024.???.01` ✏️ Runbook: rejeição, resubmit, resposta a query regulatória. *Aceite:* 3 fluxos documentados com comandos. *Entrega:* `docs/compliance/cos-workflow.md`. *Escopo:* `docs/compliance/`. *Depende:* 1.9. *Espera:* jurídico — validação do procedimento. [registro](../registro/README.md)
- **1.13** `RTSK.024.???.01` ✏️ Teste de integração: abrir → 4-eyes approve → submeter → ACCEPTED. *Aceite:* teste completo em < 30s contra mock. *Entrega:* `tests/integration/cos_workflow_test.go`. *Escopo:* `tests/integration/`. *Depende:* 1.5, 1.9. *Espera:* gate — testes com banco. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.???.01` ✏️ Rodar aceite da MS em staging. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 fluxo end-to-end em sandbox SISCOAF com resposta ACCEPTED em < 1 dia útil. *Entrega:* relatório em `registro/ms-024g/`. *Escopo:* módulo inteiro + sandbox SISCOAF. *Espera:* gate — rodada N2. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.???.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- COAF resolução / spec API SISCOAF
- `modules/compliance/domain/screening.go` (RN_FX_039)
- `docs/security/sod-matrix.md` — 4-eyes para aprovação COS
- ISO 27001 controle 5.31
- MS-024d (hits de sanctions acionam COS)

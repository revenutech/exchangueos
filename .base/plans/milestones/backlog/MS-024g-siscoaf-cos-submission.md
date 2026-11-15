# MS-024g — SISCOAF COS Submission

**Code:** `RMLS.024.014.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=14; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Compliance + Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024g/plano-original-shape-1.1.0.md](../registro/ms-024g/plano-original-shape-1.1.0.md)

## Por quê

`modules/compliance/domain/screening.go` produz `ScreeningResult.RequiresCOS=true` para hits HIGH (RN_FX_039), mas nenhum código submete o COS (Comunicação de Operação Suspeita) ao SISCOAF. Prazo regulatório é **1 dia útil** da decisão — filing manual não escala.

## Descrição

Implementar geração + submissão de XML COS para SISCOAF. Inclui workflow de gestão de caso (fila de revisão 4-eyes + aprovação do Compliance Officer antes de transmitir), XML estruturado por spec COAF, submissão mTLS, trilha de auditoria e retry/resubmit.

## Fase 1 — Implementação

### Etapa 1.1 — Aggregate + migração

- **1.1** `RTSK.024.1401.01` ✅ Aggregate `COSCase` (DRAFT → UNDER_REVIEW → APPROVED → SUBMITTED → ACCEPTED/REJECTED) + 4-eyes. *Aceite:* transições válidas aceitas, inválidas recusadas, 1 aprovador não submete. *Entrega:* `modules/compliance/domain/cos_case.go` + testes. *Escopo:* `modules/compliance/domain/`. [registro](../registro/ms-024g/2026-11-10-cos-case-4c1e.md)
- **1.2** `RTSK.024.1402.01` ✅ Migração criando `cos_cases` com lifecycle completo + audit + `payload_hash`. *Aceite:* migração up+down idempotente, FK para tenant. *Entrega:* `migrations/000011_create_cos_cases.up.sql` + `.down.sql`. *Escopo:* `migrations/`. [registro](../registro/ms-024g/2026-11-10-cos-cases-a35f.md)
- **1.3** `RTSK.024.1403.01` ✅ Migração estendendo `screening_results` com FK opcional para `cos_cases.id`. *Aceite:* FK não obrigatória, migração up+down. *Entrega:* migração em `migrations/`. *Escopo:* `migrations/`. *Depende:* `RTSK.024.1402.01`. [registro](../registro/ms-024g/2026-11-10-screening-fk-5d80.md)

### Etapa 1.2 — Workflow + wiring

- **1.14** `RTSK.024.1416.01` ✏️ Repositório de `cos_cases` com transições, versão otimista e tenant. *Aceite:* ida e volta com banco; versão velha recusada; outro tenant não lê. *Entrega:* `modules/compliance/infrastructure/` + teste com banco. *Escopo:* `modules/compliance/infrastructure/`. *Depende:* `RTSK.024.1402.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (tenant, PII). [registro](../registro/README.md)
- **1.4** `RTSK.024.1404.01` ✏️ Serviço `OpenCOSCase` acionado em hit HIGH. *Aceite:* `ScreenCounterparty` HIGH cria `COSCase` em DRAFT com payload_hash. *Entrega:* `modules/compliance/application/cos_workflow.go`. *Escopo:* `modules/compliance/application/`. *Depende:* `RTSK.024.1401.01`, `RTSK.024.1212.01`, `RTSK.024.1416.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII). [registro](../registro/README.md)
- **1.5** `RTSK.024.1405.01` ✏️ Auto-submit após sign-off do segundo aprovador. *Aceite:* 2ª aprovação transiciona para APPROVED e dispara submissão. *Entrega:* lógica em `cos_workflow.go`. *Escopo:* `modules/compliance/application/`. *Depende:* `RTSK.024.1404.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII). [registro](../registro/README.md)

### Etapa 1.3 — Fila de revisão + API

- **1.6** `RTSK.024.1406.01` ✏️ Endpoint `GET /v1/compliance/cos-cases?status=UNDER_REVIEW`. *Aceite:* retorna fila paginada com filter por status + tenant. *Entrega:* REST handler + gRPC handler sob `grpcgen`. *Escopo:* `modules/compliance/api/`. *Depende:* `RTSK.024.1404.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (tenant, PII). [registro](../registro/README.md)
- **1.7** `RTSK.024.1407.01` ✏️ Endpoint `POST /v1/compliance/cos-cases/:id/approve` (role `compliance_officer`). *Aceite:* 403 sem role; sucesso grava aprovador + timestamp. *Entrega:* handler em `modules/compliance/api/cos_grpc.go`. *Escopo:* `modules/compliance/api/`. *Depende:* `RTSK.024.1404.01`. *Espera:* dono — definição do papel `compliance_officer` no IdentityOS; revisor: IdentityOS (tenant). [registro](../registro/README.md)

### Etapa 1.4 — Submissão + marshal

- **1.8** `RTSK.024.1408.01` ✏️ Marshal COS XML por spec COAF (2026 schema). *Aceite:* round-trip valida contra XSD COAF. *Entrega:* `pkg/siscoaf/cos.go`. *Escopo:* `pkg/siscoaf/`. *Espera:* dono — confirmação da versão de schema; revisor: IdentityOS (PII). [registro](../registro/README.md)
- **1.9** `RTSK.024.1409.01` ✏️ Cliente mTLS + SOAP/REST per API SISCOAF. *Aceite:* conexão contra mock devolve protocolID; retry em 5xx. *Entrega:* `pkg/siscoaf/submitter.go`. *Escopo:* `pkg/siscoaf/`. *Depende:* `RTSK.024.1408.01`. *Espera:* dono — provisionamento da cert SISCOAF; revisor: IdentityOS (assinatura, PII). [registro](../registro/README.md)

### Etapa 1.5 — SLA + observabilidade

- **1.10** `RTSK.024.1410.01` ✏️ Alerta SLA: caso > 20h em UNDER_REVIEW (RN_FX_039 = 1 dia útil total). *Aceite:* alerta dispara em teste de injeção. *Entrega:* regra em `deploy/helm/` + métrica. *Escopo:* `deploy/helm/`, `modules/compliance/`. *Depende:* `RTSK.024.1404.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.11** `RTSK.024.1411.01` ✏️ Métricas `cos_cases_total{status}`, `cos_sla_breaches_total`, `cos_submission_duration_seconds`. *Aceite:* 3 métricas expostas. *Entrega:* instrumentação em `cos_workflow.go`. *Escopo:* `modules/compliance/`. *Depende:* `RTSK.024.1405.01`, `RTSK.024.1409.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.6 — Runbook + teste

- **1.12** `RTSK.024.1412.01` ✏️ Runbook: rejeição, resubmit, resposta a query regulatória. *Aceite:* 3 fluxos documentados com comandos. *Entrega:* `docs/compliance/cos-workflow.md`. *Escopo:* `docs/compliance/`. *Depende:* `RTSK.024.1409.01`. *Espera:* jurídico — validação do procedimento. [registro](../registro/README.md)
- **1.13** `RTSK.024.1413.01` ✏️ Teste de integração: abrir → 4-eyes approve → submeter → ACCEPTED. *Aceite:* teste completo em < 30s contra mock. *Entrega:* `tests/integration/cos_workflow_test.go`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1405.01`, `RTSK.024.1409.01`. *Espera:* gate — testes com banco. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1414.01` ✏️ Rodar aceite da MS em staging. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 fluxo end-to-end em sandbox SISCOAF com resposta ACCEPTED em < 1 dia útil. *Entrega:* relatório em `registro/ms-024g/`. *Escopo:* módulo inteiro + sandbox SISCOAF. *Espera:* gate — rodada `lote` do head de integração. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1415.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- COAF resolução / spec API SISCOAF
- `modules/compliance/domain/screening.go` (RN_FX_039)
- `docs/security/sod-matrix.md` — 4-eyes para aprovação COS
- ISO 27001 controle 5.31
- MS-024d (hits de sanctions acionam COS)

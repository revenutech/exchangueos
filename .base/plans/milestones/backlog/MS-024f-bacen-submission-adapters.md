# MS-024f — BACEN Submission Adapters (DEC + SCE-IED + SCE-Crédito + SCE-CBE)

**Code:** `RMLS.024.013.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=13; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Compliance + Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024f/plano-original-shape-1.1.0.md](../registro/ms-024f/plano-original-shape-1.1.0.md)

## Por quê

Classes de domínio para DEC + SCE-IED + SCE-Crédito + SCE-CBE existem em `modules/compliance/domain/`. Submissão real ao gateway BACEN (RDR-S / SCE / DEC) não está implementada — `bacen_reports.status` só progride a PENDING em testes. ExchangeOS real precisa enviar payloads à BACEN dentro dos prazos.

## Descrição

Construir 4 adaptadores outbound em `pkg/bacen/submission/` (um por tipo de filing) implementando o protocolo do gateway BACEN (mTLS + XML assinado + janela de submissão datada). Ligar em `modules/compliance/application` para que `SubmitBACENReport` transite PENDING → SUBMITTED → ACCEPTED/REJECTED via API real.

## Fase 1 — Implementação

### Etapa 1.1 — Cliente comum + marshal

- **1.1** `RTSK.024.1301.01` ✅ Interface `Submitter.Submit(ctx, payload)` + `Query(ctx, protocolID)`. *Aceite:* interface compila, 2 métodos documentados. *Entrega:* `pkg/bacen/submission/submitter.go`. *Escopo:* `pkg/bacen/submission/`. [registro](../registro/ms-024f/2026-11-10-submitter-9c1d.md)
- **1.2** `RTSK.024.1302.01` ✏️ Cliente mTLS com cert via Vault. *Aceite:* conexão estabelecida contra mock, 4xx/5xx tratados sem crash. *Entrega:* `pkg/bacen/submission/client.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1301.01`. *Espera:* dono — provisionamento da cert do BACEN; revisor: IdentityOS (assinatura). [registro](../registro/README.md)
- **1.3** `RTSK.024.1303.01` ☑️ Marshal `BACENReport` aggregate → XML (sem XML hand-crafted). *Aceite:* round-trip parse-marshal valida 1 fixture de cada tipo. *Entrega:* `pkg/bacen/submission/marshal.go`. *Escopo:* `pkg/bacen/submission/`. *Espera:* gate — N2 do lote 9.2; revisor: IdentityOS (PII). [registro](../registro/ms-024f/2026-11-10-marshal-7b2a.md)
- **1.4** `RTSK.024.1304.01` ✏️ Assinatura XML XAdES-BES por spec técnica BACEN. *Aceite:* XML assinado validado por verificador externo (xmlsectool). *Entrega:* seção em `client.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1303.01`. *Espera:* dono — definição do certificado de assinatura; revisor: IdentityOS (assinatura). [registro](../registro/README.md)
- **1.5** `RTSK.024.1305.01` ✏️ Retry exponencial + circuit breaker (5xx > 60s → open). *Aceite:* 3 retries em 5xx transiente; breaker abre e fecha conforme política. *Entrega:* seção em `client.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1302.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Adaptadores por tipo

- **1.6** `RTSK.024.1306.01` ✏️ Submitter DEC (Lei 14.286/2021, > USD 10K). *Aceite:* 1 payload DEC submetido ao mock devolve protocolID. *Entrega:* `pkg/bacen/submission/dec.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1301.01`, `RTSK.024.1303.01`, `RTSK.024.1304.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII) + LedgerOS (dinheiro). [registro](../registro/README.md)
- **1.7** `RTSK.024.1307.01` ✏️ Submitter SCE-IED (registro em 30 dias). *Aceite:* payload SCE-IED aceito pelo mock, janela de 30 dias respeitada. *Entrega:* `pkg/bacen/submission/sce_ied.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1301.01`, `RTSK.024.1303.01`, `RTSK.024.1304.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII) + LedgerOS (dinheiro). [registro](../registro/README.md)
- **1.8** `RTSK.024.1308.01` ✏️ Submitter SCE-Crédito (sucessor RDE-ROF). *Aceite:* payload aceito pelo mock. *Entrega:* `pkg/bacen/submission/sce_credito.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1301.01`, `RTSK.024.1303.01`, `RTSK.024.1304.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII) + LedgerOS (dinheiro). [registro](../registro/README.md)
- **1.9** `RTSK.024.1309.01` ✏️ Submitter SCE-CBE. *Aceite:* payload aceito pelo mock. *Entrega:* `pkg/bacen/submission/sce_cbe.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1301.01`, `RTSK.024.1303.01`, `RTSK.024.1304.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII) + LedgerOS (dinheiro). [registro](../registro/README.md)

### Etapa 1.3 — Infra + wiring

- **1.10** `RTSK.024.1310.01` ✏️ `SubmitBACENReport` ligado ao submitter real via DI. *Aceite:* teste com mock valida PENDING → SUBMITTED. *Entrega:* `service.go`. *Escopo:* `modules/compliance/application/`. *Depende:* `RTSK.024.1306.01`, `RTSK.024.1307.01`, `RTSK.024.1308.01`, `RTSK.024.1309.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII) + LedgerOS (dinheiro). [registro](../registro/README.md)
- **1.11** `RTSK.024.1311.01` ✏️ Network policy / sidecar Helm para egress BACEN (IP whitelisted). *Aceite:* manifesto aplica, teste de conectividade valida. *Entrega:* `deploy/k8s/network-policies/bacen-egress.yaml`. *Escopo:* `deploy/k8s/`. *Espera:* Baremetal — whitelist do IP BACEN. [registro](../registro/README.md)

### Etapa 1.4 — Observabilidade + runbook

- **1.12** `RTSK.024.1312.01` ✏️ Métricas `bacen_submission_total{type,status}`, `bacen_submission_duration_seconds{type}`, `bacen_submission_failed_total{type,reason}`. *Aceite:* 3 métricas expostas, series por tipo. *Entrega:* instrumentação em `client.go`. *Escopo:* `pkg/bacen/submission/`. *Depende:* `RTSK.024.1305.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.13** `RTSK.024.1313.01` ✏️ Runbook: rejeições comuns + como emendar + resubmeter. *Aceite:* runbook cobre os 5 motivos de rejeição mais citados pela BACEN. *Entrega:* `docs/operations/bacen-runbook.md`. *Escopo:* `docs/operations/`. *Depende:* `RTSK.024.1310.01`. *Espera:* jurídico — validação do procedimento de emenda. [registro](../registro/README.md)

### Etapa 1.5 — Testes

- **1.14** `RTSK.024.1314.01` ✅ Mock BACEN gateway em `tests/integration/bacen_mock_server.go` para CI. *Aceite:* mock aceita DEC/SCE-* e devolve protocolID + status. *Entrega:* `tests/integration/bacen_mock_server.go`. *Escopo:* `tests/integration/`. [registro](../registro/ms-024f/2026-11-10-bacen-mock-d2e6.md)
- **1.15** `RTSK.024.1315.01` ✏️ Teste de integração: submeter 1 de cada tipo → poll → ACCEPTED. *Aceite:* 4 submissões bem-sucedidas em < 15s total. *Entrega:* `tests/integration/bacen_submission_test.go`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1310.01`, `RTSK.024.1314.01`. *Espera:* gate — testes com banco. [registro](../registro/README.md)
- **1.16** `RTSK.024.1318.01` ✏️ `GoString()` e JSON redigido no `Payload`. *Aceite:* `%#v` e `json.Marshal` não imprimem o Body; teste com controle negativo. *Entrega:* `pkg/bacen/submission/submitter.go` + teste. *Escopo:* `pkg/bacen/submission/`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII). [registro](../registro/README.md)
- **1.17** `RTSK.024.1319.01` ✏️ Gravar `reference_date` como data civil de Brasília. *Aceite:* relatório criado às 22h30 BRT fica com o dia de Brasília na coluna DATE; teste com banco. *Entrega:* `modules/compliance/infrastructure/postgres/repos.go` + teste. *Escopo:* `modules/compliance/`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII). [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1316.01` ✏️ Rodar aceite da MS em staging. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 submissão real a sandbox BACEN de cada tipo com resposta ACCEPTED em < 60s. *Entrega:* relatório em `registro/ms-024f/`. *Escopo:* módulo inteiro + staging + sandbox BACEN. *Espera:* gate — rodada N2. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1317.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- Lei 14.286/2021 + Res BCB 277-561
- `modules/compliance/domain/bacen_report.go`
- Padrão marshal em `pkg/iso20022/`
- ISO 27001 controle 5.31 (legal/statutory/regulatory)
- MS-024e (códigos de natureza alimentam DEC)
- MS-024g (filing SISCOAF paralelo)

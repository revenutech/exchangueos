# MS-024d — Live Sanctions Providers (OFAC / UN / EU / COAF)

**Code:** `RMLS.024.012.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=12; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Compliance + Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024d/plano-original-shape-1.1.0.md](../registro/ms-024d/plano-original-shape-1.1.0.md)

## Por quê

A triagem atual usa stub in-process que devolve zero matches. As 4 listas autoritativas (OFAC SDN, UN 1267 Consolidated, EU restrictive measures, COAF) devem ser baixadas, parseadas, normalizadas e consultáveis com fuzzy matching. Sem isto, triagem é teatro.

## Descrição

Implementar 4 providers por trás de uma interface comum `SanctionsProvider` + cache quente + cron de refresh. Cada provider parseia seu formato nativo (OFAC XML, UN XML, EU CSV, COAF XLSX), normaliza em `SanctionsEntry` canônico e expõe `Match(name, doc, country)` com threshold configurável (Jaro-Winkler 0.85 default).

## Fase 1 — Implementação

### Etapa 1.1 — Interface + cache

- **1.1** `RTSK.024.1201.01` ✅ Interface `SanctionsProvider` com `Refresh(ctx)` + `Match(query MatchQuery)`. *Aceite:* interface compila, godoc completa. *Entrega:* `pkg/sanctions/provider.go`. *Escopo:* `pkg/sanctions/`. [registro](../registro/ms-024d/2026-11-10-sanctions-provider-61f0.md)
- **1.2** `RTSK.024.1202.01` ✏️ Cache in-memory com inverted index + scorer Jaro-Winkler. *Aceite:* 100k entradas indexadas em < 2s, Match p99 < 10ms. *Entrega:* `pkg/sanctions/cache.go`, `pkg/sanctions/jarowinkler.go` (ou import `github.com/xrash/smetrics`). *Escopo:* `pkg/sanctions/`. *Depende:* `RTSK.024.1201.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Providers (4 fontes)

- **1.3** `RTSK.024.1203.01` ✏️ Provider OFAC (XML SDN). *Aceite:* parse round-trip contra fixture 2026-Q2 + golden test passa. *Entrega:* `pkg/sanctions/ofac/provider.go`. *Escopo:* `pkg/sanctions/ofac/`. *Depende:* `RTSK.024.1201.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.4** `RTSK.024.1204.01` ✏️ Provider UN (XML 1267). *Aceite:* parse + golden test. *Entrega:* `pkg/sanctions/un/provider.go`. *Escopo:* `pkg/sanctions/un/`. *Depende:* `RTSK.024.1201.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.5** `RTSK.024.1205.01` ✏️ Provider EU (CSV restrictive measures). *Aceite:* parse + golden test. *Entrega:* `pkg/sanctions/eu/provider.go`. *Escopo:* `pkg/sanctions/eu/`. *Depende:* `RTSK.024.1201.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.6** `RTSK.024.1206.01` ✏️ Provider COAF (XLSX). *Aceite:* parse + golden test. *Entrega:* `pkg/sanctions/coaf/provider.go`. *Escopo:* `pkg/sanctions/coaf/`. *Depende:* `RTSK.024.1201.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Refresh + freshness

- **1.7** `RTSK.024.1207.01` ✏️ Binário `cmd/sanctions-refresher/` atualiza 4 listas em stagger horário. *Aceite:* binário aceita `--providers all|ofac|un|eu|coaf`, executa refresh e logs duração. *Entrega:* `main.go`. *Escopo:* `cmd/sanctions-refresher/`. *Depende:* `RTSK.024.1203.01`, `RTSK.024.1204.01`, `RTSK.024.1205.01`, `RTSK.024.1206.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.8** `RTSK.024.1208.01` ✏️ SLO de frescor: alerta se lista > 24h sem refresh. *Aceite:* métrica `sanctions_list_age_seconds{list}` exposta, alerta ao ultrapassar 86400. *Entrega:* instrumentação + regra em `deploy/helm/`. *Escopo:* `cmd/sanctions-refresher/`, `deploy/helm/`. *Depende:* `RTSK.024.1207.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.9** `RTSK.024.1209.01` ✏️ Fallback para snapshot cacheado quando upstream inalcançável (max 24h stale). *Aceite:* indisponibilidade de upstream não derruba Match; snapshot > 24h bloqueia e alerta. *Entrega:* lógica em `cmd/sanctions-refresher/`. *Escopo:* `cmd/sanctions-refresher/`. *Depende:* `RTSK.024.1207.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.10** `RTSK.024.1210.01` ✏️ Helm CronJob `sanctions-refresher-cronjob.yaml` (hourly). *Aceite:* manifesto renderiza, agenda aplica. *Entrega:* `deploy/helm/exchangeos/templates/sanctions-refresher-cronjob.yaml`. *Escopo:* `deploy/helm/exchangeos/`. *Depende:* `RTSK.024.1207.01`. *Espera:* Baremetal — pipeline do Helm. [registro](../registro/README.md)

### Etapa 1.4 — Auditoria + integração compliance

- **1.11** `RTSK.024.1211.01` ✏️ Logging de hits (inclusive sub-threshold) como audit event para fluxo COS. *Aceite:* todo match gera `audit_event(type='SANCTIONS_HIT', ...)` com score + fonte. *Entrega:* instrumentação em `pkg/sanctions/cache.go`. *Escopo:* `pkg/sanctions/`, `pkg/audit/`. *Depende:* `RTSK.024.1202.01`. *Espera:* DPO — revisão de retenção de audit; revisor: IdentityOS (PII). [registro](../registro/README.md)
- **1.12** `RTSK.024.1212.01` ✏️ Compliance `ScreenCounterparty` consome providers reais via DI. *Aceite:* chamada bate em 4 providers, hit maior score ganha. *Entrega:* `modules/compliance/application/service.go` atualizado. *Escopo:* `modules/compliance/application/`. *Depende:* `RTSK.024.1202.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII). [registro](../registro/README.md)

### Etapa 1.5 — Documentação + ameaça

- **1.13** `RTSK.024.1213.01` ✏️ Documentar vendor + URL + método de auth por lista. *Aceite:* `docs/security/sanctions-sources.md` com 4 seções e credencial em Vault path. *Entrega:* `docs/security/sanctions-sources.md`. *Escopo:* `docs/security/`. *Depende:* `RTSK.024.1203.01`, `RTSK.024.1204.01`, `RTSK.024.1205.01`, `RTSK.024.1206.01`. *Espera:* dono — definição de contratos com vendors. [registro](../registro/README.md)
- **1.14** `RTSK.024.1214.01` ✏️ STRIDE: adicionar ameaça de list-poisoning + verificação (fontes assinadas). *Aceite:* modelo atualizado com mitigação (checksum ou signature verify). *Entrega:* `docs/security/threat-model.md`. *Escopo:* `docs/security/`. *Depende:* `RTSK.024.1213.01`. *Espera:* dono — definição de signatário confiável. [registro](../registro/README.md)
- **1.15** `RTSK.024.1215.01` ✏️ Teste de integração contra fixture XML/CSV/XLSX capturados 2026-Q2. *Aceite:* 4 providers parseiam fixture e Match encontra entry conhecido. *Entrega:* `tests/integration/sanctions_test.go`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1203.01`, `RTSK.024.1204.01`, `RTSK.024.1205.01`, `RTSK.024.1206.01`, `RTSK.024.1212.01`. *Espera:* gate — testes sem banco. [registro](../registro/README.md)
- **1.16** `RTSK.024.1218.01` ✏️ `GoString()` e JSON redigido no `MatchQuery`. *Aceite:* `%#v` e `json.Marshal` não imprimem o Name; teste com controle negativo. *Entrega:* `pkg/sanctions/provider.go` + teste. *Escopo:* `pkg/sanctions/`. *Espera:* sessão dona — implementação; revisor: IdentityOS (PII). [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1216.01` ✏️ Rodar aceite da MS em staging. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 ciclo horário de refresh em staging sem falha + 1 hit conhecido reproduzido por fixture. *Entrega:* relatório em `registro/ms-024d/`. *Escopo:* módulo inteiro + staging. *Espera:* gate — rodada N2. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1217.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `docs/integrations/complos.md` — substituição eventual por ComplOS
- `modules/compliance/domain/screening.go` — RN_FX_039 SISCOAF COS
- ISO 27001 controle 5.7 (threat intelligence)
- BACEN Circ 3.978 (CCS PLD-FT)

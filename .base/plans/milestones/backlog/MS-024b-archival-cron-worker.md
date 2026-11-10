# MS-024b — Archival Cron Worker

**Code:** `RMLS.024.010.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=10; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024b/plano-original-shape-1.1.0.md](../registro/ms-024b/plano-original-shape-1.1.0.md)

## Por quê

`docs/security/data-lifecycle/archival-cron.md` especifica `cmd/archiver` (02:00 SP nightly) + tabela `archive_policy` + escrita em GCS Coldline com manifesto SHA256, mas sem código. Até construir, o CRDB retém tudo para sempre.

## Descrição

Implementar `cmd/archiver/main.go` como CronJob Kubernetes que envelhece linhas do CRDB para GCS Coldline conforme política dirigida por dados. Inclui escritor parquet + manifesto SHA256 + hard-delete pós-retenção + emissão de evento no outbox + harness trimestral de ensaio de restauração.

## Fase 1 — Implementação

### Etapa 1.1 — Schema + política

- **1.1** `RTSK.024.1001.01` ⬜ Criar tabelas `archive_policy` e `archive_manifest`. *Aceite:* migração 000010 up+down idempotente, seed com 25 linhas (uma por tabela de `data-lifecycle/README.md`). *Entrega:* `migrations/000010_create_archive_tables.up.sql` + `.down.sql`. *Escopo:* `migrations/`. *Espera:* dono — liberar código (plano da etapa 9); revisor: IdentityOS (PII: retenção LGPD). [registro](../registro/README.md)
- **1.2** `RTSK.024.1002.01` ✏️ Loader + cache de `archive_policy`. *Aceite:* carrega 25 políticas, invalida cache em UPDATE. *Entrega:* `internal/archive/policy.go`. *Escopo:* `internal/archive/`. *Depende:* `RTSK.024.1001.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Escritor parquet + GCS

- **1.3** `RTSK.024.1003.01` ✏️ Escrita bulk-parquet em GCS com zstd. *Aceite:* batch escrito em `gs://exchangeos-archive-<env>/<prefix>/YYYY/MM/DD/batch-<uuid>.parquet.gz`, leitura round-trip valida. *Entrega:* `internal/archive/writer.go`. *Escopo:* `internal/archive/`. *Depende:* `RTSK.024.1002.01`. *Espera:* dono — provisionamento do bucket GCS; revisor: IdentityOS (PII: cópia ao GCS). [registro](../registro/README.md)
- **1.4** `RTSK.024.1004.01` ✏️ Computar SHA256 e gravar em `archive_manifest`. *Aceite:* hash do arquivo escrito bate com o persistido. *Entrega:* `internal/archive/repository.go`. *Escopo:* `internal/archive/`. *Depende:* `RTSK.024.1003.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Marcação e exclusão

- **1.5** `RTSK.024.1005.01` ✏️ Marcar linhas `archived_at` após upload. *Aceite:* `UPDATE <table> SET archived_at = now()` apenas no batch atual, sem race. *Entrega:* lógica em `repository.go`. *Escopo:* `internal/archive/`. *Depende:* `RTSK.024.1004.01`. *Espera:* sessão dona — implementação; revisor: LedgerOS (dinheiro: trades arquivados). [registro](../registro/README.md)
- **1.6** `RTSK.024.1006.01` ✏️ Hard-delete após `retention_after` para tabelas sem legal-hold. *Aceite:* linhas com idade > política e sem flag são removidas; legal-hold permanece. *Entrega:* lógica em `repository.go`. *Escopo:* `internal/archive/`. *Depende:* `RTSK.024.1005.01`. *Espera:* DPO — revisão da política de retenção; revisor: IdentityOS (PII) + LedgerOS (dinheiro). [registro](../registro/README.md)

### Etapa 1.4 — Observabilidade + evento

- **1.7** `RTSK.024.1007.01` ✏️ Emissão de `archival.batch_completed.v1` no outbox. *Aceite:* evento emitido uma vez por batch, com `batch_id`, `table`, `row_count`, `gcs_uri`, `sha256`. *Entrega:* publisher em `internal/archive/writer.go`. *Escopo:* `internal/archive/`, `pkg/outbox/`. *Depende:* `RTSK.024.1004.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.8** `RTSK.024.1008.01` ✏️ Métricas: `archive_jobs_completed_total`, `archive_batch_rows`, `archive_batch_bytes`, `archive_pending_rows{table}`. *Aceite:* 4 métricas expostas em `/metrics`. *Entrega:* instrumentação em `cmd/archiver/`. *Escopo:* `cmd/archiver/`. *Depende:* `RTSK.024.1007.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.5 — Binário + deploy

- **1.9** `RTSK.024.1009.01` ✏️ `cmd/archiver/main.go` com flags `--policy-table`, `--batch-size`, `--max-tables`. *Aceite:* binário aceita flags, orquestra políticas 1-N. *Entrega:* `cmd/archiver/main.go`. *Escopo:* `cmd/archiver/`. *Depende:* `RTSK.024.1002.01`, `RTSK.024.1003.01`, `RTSK.024.1005.01`, `RTSK.024.1006.01`, `RTSK.024.1007.01`, `RTSK.024.1008.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.10** `RTSK.024.1010.01` ✏️ Helm CronJob `archiver-cronjob.yaml` rodando 05:00 UTC (= 02:00 SP). *Aceite:* manifesto renderiza, agenda aplica. *Entrega:* `deploy/helm/exchangeos/templates/archiver-cronjob.yaml`. *Escopo:* `deploy/helm/exchangeos/`. *Depende:* `RTSK.024.1009.01`. *Espera:* Baremetal — pipeline do Helm. [registro](../registro/README.md)

### Etapa 1.6 — Restauração e observabilidade externa

- **1.11** `RTSK.024.1011.01` ✏️ Harness `scripts/archive-restore-test.sh` para ensaio trimestral. *Aceite:* script restaura 1 batch de fixture e valida contagem de linhas + SHA256. *Entrega:* `scripts/archive-restore-test.sh`. *Escopo:* `scripts/`. *Depende:* `RTSK.024.1004.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.12** `RTSK.024.1012.01` ✏️ Teste de integração archive → manifest → marcação → hard-delete. *Aceite:* teste passa contra CRDB testcontainer. *Entrega:* `tests/integration/archiver_test.go`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1006.01`, `RTSK.024.1011.01`. *Espera:* gate — testes com banco. [registro](../registro/README.md)
- **1.13** `RTSK.024.1013.01` ✏️ Painel Grafana no dashboard de FinOps (crescimento de storage + projeção de custo). *Aceite:* painel renderiza com dados das métricas de 1.8. *Entrega:* JSON do painel em `deploy/grafana/`. *Escopo:* `deploy/grafana/`. *Depende:* `RTSK.024.1008.01`. *Espera:* dono — definição da fórmula de projeção. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1014.01` ✏️ Rodar aceite da MS em staging. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 ciclo noturno completo em staging com batch > 0 + restauração de 1 batch bem sucedida. *Entrega:* relatório em `registro/ms-024b/`. *Escopo:* módulo inteiro + staging. *Espera:* gate — rodada N2. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1015.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `docs/security/data-lifecycle/archival-cron.md` — especificação original
- `docs/security/data-lifecycle/README.md` — cronograma de retenção por tabela
- `docs/operations/cost-allocation.md` — rotulagem do bucket GCS
- ISO 27001 controles 8.10, 8.13

# MS-024i — CRUD Test Suite Completion

**Code:** `RMLS.024.015.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=15; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** All BC owners + Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024i/plano-original-shape-1.1.0.md](../registro/ms-024i/plano-original-shape-1.1.0.md)

## Por quê

MS-023s (Local deploy + ~271 CRUD tests em 14 BCs) foi entregue como catálogo + harness, não como 271 testes individualmente escritos. Cobertura de domain é sólida (≥ 80% por coverage gate), mas cobertura de persistence + integração cross-aggregate é desigual.

## Descrição

Escrever os testes CRUD de integração remanescentes por BC × postgres backend, elevando cobertura de integração a ≥ 70% e travando o catálogo de MS-023s. Cada teste roda contra schema per-test em CRDB testcontainers para ficar parallel-safe.

## Fase 1 — Implementação

### Etapa 1.1 — Matriz CRUD por BC

- **1.1** `RTSK.024.1501.01` ⬜ Matriz 14 BCs × 8 ops (Create/Read/Update/Delete/List/Filter/Page/Tx-rollback) = 112 testes. *Aceite:* 112 testes verdes contra testcontainer CRDB, um arquivo por BC, piso 80/70 nos 11 módulos hoje abaixo dele. *Entrega:* `modules/*/infrastructure/postgres/*_test.go`. *Escopo:* `modules/*/infrastructure/postgres/`. *Espera:* dono — liberar código (plano da etapa 9). [registro](../registro/README.md)
  - Piso em `23fca30` (11 módulos abaixo, piso domain ≥ 80, application ≥ 70): admin/domain 73.0, cfets_capture/domain 71.2, cfets_confirmation/domain 60.8, cls_settlement/domain 75.3, compliance/domain 67.3, netreport/domain 72.0, payin/domain 71.2, quote/domain 70.2, refdata/domain 68.3, trade/application 56.4, trade/domain 67.2.
- **1.2** `RTSK.024.1502.01` ✏️ Isolamento por teste via `CREATE SCHEMA tenant_test_<id>`. *Aceite:* 10 testes paralelos não colidem em schema. *Entrega:* helper em `tests/testutil/`. *Escopo:* `tests/testutil/`, `modules/*/`. *Depende:* `RTSK.024.1501.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (tenant). [registro](../registro/README.md)

### Etapa 1.2 — Concorrência + sagas

- **1.3** `RTSK.024.1503.01` ✏️ 14 testes de bump de versão otimista (1 por BC). *Aceite:* race detectado reverte um dos commits; 14 testes verdes. *Entrega:* `tests/integration/concurrency_test.go`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1501.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.4** `RTSK.024.1504.01` ✏️ 5 sagas cross-aggregate (Trade-Quote, Trade-CLS-Settlement, Trade-Compliance, EOD-Position, CLS-PayIn-NetReport). *Aceite:* 5 fluxos nomeados passam. *Entrega:* `tests/integration/sagas/*_test.go` (5 arquivos). *Escopo:* `tests/integration/sagas/`. *Depende:* `RTSK.024.1501.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Gates + CI

- **1.5** `RTSK.024.1505.01` ✏️ Tag `//go:build integration` + testcontainers-go CRDB em todos os testes novos. *Aceite:* `go test ./tests/integration/...` com e sem `-tags integration` tem comportamento esperado. *Entrega:* build tags em todos os `_test.go`. *Escopo:* `tests/integration/`, `modules/*/infrastructure/postgres/`. *Depende:* `RTSK.024.1501.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.6** `RTSK.024.1506.01` ✏️ CI task `tests:integration` roda matriz em < 15 min. *Aceite:* pipeline verde em staging CI, duração mensurada. *Entrega:* Taskfile target + workflow. *Escopo:* `Taskfile.yml`, `.github/workflows/`. *Depende:* `RTSK.024.1505.01`. *Espera:* Baremetal — runner CI. [registro](../registro/README.md)
- **1.7** `RTSK.024.1507.01` ✏️ Cobertura de integração ≥ 70%. *Aceite:* relatório de coverage mostra ≥ 70% no layer de integração. *Entrega:* configuração em `.golangci.yml` + CI gate. *Escopo:* `.golangci.yml`. *Depende:* `RTSK.024.1501.01`, `RTSK.024.1503.01`, `RTSK.024.1504.01`. *Espera:* gate — ratchet de coverage. [registro](../registro/README.md)
- **1.11** `RTSK.024.1513.01` ✅ `.ci-local/escopo.conf` com as 6 categorias sensíveis e controle negativo. *Aceite:* cada categoria dá «amplia» no `--nivel` do gate; commit fora dá «N1». *Entrega:* `escopo.conf` + controle negativo em `tests/ci-local/`. *Escopo:* `.ci-local/`. [registro](../registro/ms-024i/2026-11-10-escopo-conf-5e1a.md)
- **1.12** `RTSK.024.1514.01` ✅ Subir `golang.org/x/net` para v0.62.0 (CVE-2026-78661 HIGH em v0.60.0, trivy reprova). *Aceite:* `go list -m golang.org/x/net` devolve `v0.62.0`; `go build ./...` e `go test -race -count=1 ./...` verdes. *Entrega:* `go.mod` + `go.sum`. *Escopo:* `go.mod`, `go.sum`. [registro](../registro/ms-024i/2026-11-10-xnet-0-62-0-a7c3.md)
- **1.13** `RTSK.024.1515.01` ✏️ Tirar as referências ao hub `cockroachdb` removido em 10/11. *Aceite:* `CRDB_HUB` e `../cockroachdb` fora do `Taskfile.yml` e do `.claude/settings.json`. *Entrega:* os dois arquivos. *Escopo:* `Taskfile.yml`, `.claude/settings.json`. *Espera:* dono — liberar código e palavra sobre o `.claude/settings.json`. [registro](../registro/README.md)
- **1.14** `RTSK.024.1516.01` ⬜ Ajustar o `.ci-local/escopo.conf`. *Aceite:* dinheiro em `pkg/bacen/` só no pacote raiz (`iof.go`), não em `submission/` nem `codegen/`; 000010 em pii, 000011 e 000014 em tenant e pii; controle negativo cobre os casos. *Entrega:* `escopo.conf` + teste. *Escopo:* `.ci-local/`, `tests/ci-local/`. *Espera:* sessão dona — implementação; revisor: Baremetal (escopo). [registro](../registro/README.md)
- **1.15** `RTSK.024.1517.01` ⬜ Piso Go 1.26.11 (GO-2026-6859 em `net/http`, govulncheck reprova o 1.26.10). *Aceite:* `go.mod`, workflows e Dockerfiles em 1.26.11; `mudanca` (N1) verde com govulncheck limpo. *Entrega:* commit próprio na develop, antes dos ramos de lote. *Escopo:* `go.mod`, `.github/workflows/`, `docker/`. *Espera:* gate — `mudanca` na fila; dono — merge. [registro](../registro/README.md)
- **1.16** `RTSK.024.1518.01` ⬜ Remover do `env` do `.claude/settings.json` o `EXCHANGEOS_PROJECT_ROOT` (caminho antigo `revenu-tech/…`) e o `GO_VERSION` 1.25; nada os lê (dono, 2026-11-12, opção A). *Aceite:* as duas chaves e a menção no `.claude/README.md` fora; JSON válido; nada mais muda. *Entrega:* ramo `chore/ms-024i-1.16-env`. *Escopo:* `.claude/`. *Espera:* sessão dona — registro (prova `mudanca` 11f2a19, merge 9a96f2b). [registro](../registro/README.md)
- **1.17** `RTSK.024.1519.01` ✏️ Decidir o `GO_VERSION: 1.25.1` do `Taskfile.yml:8` (nenhuma task o lê; piso 1.26.11). *Aceite:* a variável sai ou passa a 1.26.11, com o motivo no commit; `task --list` segue funcionando. *Entrega:* diff do `Taskfile.yml`. *Escopo:* `Taskfile.yml`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.18** `RTSK.024.1520.01` ✏️ Lista dos testes `vivo` (dono, 2026-11-12): `tests/integration/admin_crud_test.go`, 6 testes contra o servidor em :8084, atrás de `integration && vivo`. *Aceite:* a lista bate com `git grep -l 'go:build.*vivo'`; cada item roda à mão com `-tags "integration vivo"`. *Entrega:* esta linha. *Escopo:* `tests/`. *Espera:* sessão dona — rodar à mão. [registro](../registro/README.md)
- **1.19** `RTSK.024.1521.01` ✏️ Roundtrips XSD sem falso verde (dono, 13/11): o gate roda `scripts/download-xsd.sh` antes de `pkg/iso20022`; os roundtrips camt.088 e fxtr.013 reprovam, não saltam, sem o XSD. *Aceite:* a `mudanca` (N1) mostra `--- PASS` nos dois roundtrips; sem XSD, reprova. *Entrega:* passo + testes. *Escopo:* `.ci-local/`, `pkg/iso20022/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.20** `RTSK.024.1522.01` ✏️ `admin/infrastructure/postgres` vermelho na `migracoes` (0c903bd, 13/11): 3 testes do `EODJobRepo` caem em `eod_job.save: null value in column "steps_done" violates not-null constraint (23502)`. *Aceite:* o pacote passa na `migracoes`. *Entrega:* repo ou teste corrigido. *Escopo:* `modules/admin/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.21** `RTSK.024.1523.01` ✏️ `compliance/infrastructure/postgres` vermelho na `migracoes` (0c903bd, 13/11): `TestScreeningRepo_AppendOnly` cai em `screening.save: null value in column "hits" violates not-null constraint (23502)`. *Aceite:* o pacote passa na `migracoes`. *Entrega:* repo ou teste corrigido. *Escopo:* `modules/compliance/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.22** `RTSK.024.1524.01` ✏️ `tests/integration` vermelho na `migracoes` (0c903bd, 13/11): `TestErasureExecutor_Apply_RedactAndDelete` cai em `op[1] actors: duplicate key value violates unique constraint "actors_tenant_id_external_sub_key" (23505)`. *Aceite:* o teste passa na `migracoes`. *Entrega:* correção. *Escopo:* `tests/integration/`. *Espera:* dono — abrir a sessão Revenu Common, revisora PII do redact. [registro](../registro/README.md)
- **1.23** `RTSK.024.1525.01` ✏️ Regras do fluxo de teste (dono, 15/11, auditoria): por tarefa `fluxo`/`mudanca`; `lote` só quando o lote entra na integração; `completa` uma vez no fim da linha; head com prova verde não reroda, cita a prova; 3 vermelhos seguidos: muda o código antes de entrar. *Aceite:* as rodadas seguem as regras. *Entrega:* esta linha. *Escopo:* `.ci-local/`. *Espera:* sessão dona — aplicar. [registro](../registro/README.md)
- **1.24** `RTSK.024.1526.01` ✏️ Commit só de documentação ou plano não abre gate pesado: roda só o base-lint, na pista leve; dono na fila `exchangeos/<MS>-<tarefa>/<modo>` (dono, 15/11). *Aceite:* `lista` num commit só de `.base/` mostra só o base-lint. *Entrega:* `passos.conf` + nome na fila. *Escopo:* `.ci-local/`. *Espera:* sessão Baremetal — R1 e R2 do MS-023. [registro](../registro/README.md)
- **1.25** `RTSK.024.1527.01` ✏️ Uma passada `go test -json -race -cover` por pacote, em paralelo; ratchet e cobertura leem os JSONs em vez de refazer `go test ./...` (dono, 15/11). *Aceite:* no `lote` cada pacote roda `go test` uma vez; ratchet e cobertura dão o mesmo veredito de antes no mesmo head. *Entrega:* `passos.conf`. *Escopo:* `.ci-local/`. *Espera:* sessão Baremetal — R1 e R2 do MS-023. [registro](../registro/README.md)
- **1.26** `RTSK.024.1528.01` ✏️ Chave estreita nos passos globais (hoje com chave `.`, rodam em todo commit) (dono, 15/11). *Aceite:* um commit só de plano não dispara esses passos; um diff de código ainda dispara cada um; nenhum passo sai do escopo que cobre. *Entrega:* `passos.conf`. *Escopo:* `.ci-local/`. *Espera:* sessão Baremetal — R1 e R2 do MS-023. [registro](../registro/README.md)
- **1.27** `RTSK.024.1529.01` ✏️ Rename catraca → ratchet, só nome (dono, 15/11): scripts, passos e `CATRACA_BASE` (fica de apelido até o fim da linha), junto da 1.25/1.26; registro e linha fechada ficam. *Aceite:* `git grep -i catraca` só no registro e no apelido; mesmo veredito no mesmo head. *Entrega:* diff. *Escopo:* `.ci-local/`, `scripts/ci-local/`, `tests/ci-local/`. *Espera:* sessão Baremetal — R1 e R2 do MS-023. [registro](../registro/README.md)
- **1.28** `RTSK.024.1530.01` ✏️ R10 (RN_TST_001, dono 15/11): os 3 ratchets (`events-baseline.txt`, `catraca.sh` e o passo) citam `RPTN.091.026.01`; a linha de base diz `# ratchet: teto`; o `events-baseline` ganha controle negativo que o cita. Junto da 1.27. *Aceite:* `ratchet_lint.py --repos exchangeos` dá 0/0/0. *Entrega:* diff. *Escopo:* `.base/`, `.ci-local/`, `scripts/`, `tests/`. *Espera:* sessão Baremetal — 1.62/1.63 do MS-023. [registro](../registro/README.md)

### Etapa 1.4 — Documentação + política

- **1.8** `RTSK.024.1508.01` ✏️ Atualizar `tests/integration/README.md` com catálogo do que está coberto + fora de escopo. *Aceite:* README lista 131+ testes por BC + out-of-scope. *Entrega:* `tests/integration/README.md`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1501.01`, `RTSK.024.1502.01`, `RTSK.024.1503.01`, `RTSK.024.1504.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.9** `RTSK.024.1509.01` ✏️ Política de teste flaky: 2 falhas consecutivas → skip + issue + owner em 24h. *Aceite:* doc existe, processo descrito. *Entrega:* `docs/engineering/flaky-test-policy.md`. *Escopo:* `docs/engineering/`. *Espera:* dono — definição do processo. [registro](../registro/README.md)
- **1.10** `RTSK.024.1510.01` ✏️ Lefthook pre-push rodando testes de integração com `-short`. *Aceite:* pre-push dispara subset rápido antes de permitir push. *Entrega:* atualização em `lefthook.yml`. *Escopo:* `lefthook.yml`. *Depende:* `RTSK.024.1505.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1511.01` ✏️ Rodar aceite da MS. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 rodada `lote` (N2) verde + coverage report ≥ 70% arquivado. *Entrega:* relatório em `registro/ms-024i/`. *Escopo:* módulo inteiro. *Espera:* gate — rodada `lote` do head de integração. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1512.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- MS-023s (entregou catálogo + harness que este fecha)
- `.claude/rules/tests.md` — convenções testify
- Coverage gate em `.golangci.yml`
- ISO 27001 controle 8.29 (security testing)

# Catálogo de padrões — exchangeos

Padrões (`RPTN`) deste repositório. Os transversais vivem no catálogo da plataforma, [`cognee-revenu/.base/patterns/plataforma/`](../../../cognee-revenu/.base/patterns/plataforma/index.md) (domínio 091, ADR-018 da plataforma), e citam-se aqui qualificados: `cognee-revenu:PTN-NNN`. Os padrões próprios do módulo, quando existirem, entram na primeira tabela com o código do domínio do módulo (registro de domínios no `baremetal/.base/plans/CONVENTIONS.md` § 3).

**Precedência (cognee-revenu:ADR-019):** fundação → ontologia → ADR → **padrão** → regra → fluxo. Um padrão descreve; quem obriga é a regra (cognee-revenu:RN_CTX_027). A correção de um defeito em direção a um padrão está no catálogo `RFIX` (ledgeros, domínio 053), fora da precedência (cognee-revenu:O162).

## Padrões próprios do módulo

| Alias | Code | Título | Tema | Estado | ADR que adota | Exemplo |
|-------|------|--------|------|--------|---------------|---------|

Nenhum ainda.

## Padrões da plataforma que este repositório aplica

Medido em `origin/develop` = `6d23578`, 2026-10-24 BRT (MS-018 6.2). Alcance: verificado por go list -deps ./cmd/.... **sim** = o pacote do sinal é alcançável a partir de `cmd/` e o símbolo tem chamador de produção alcançável; **parcial** = há código, mas sem chamador alcançável (órfão, só teste ou fora de `cmd/`). O sinal é procurado por símbolo e por padrão de texto: a evidência mostra onde está, e o dono do módulo confere se é a aplicação do padrão. O `PTN-008` é um padrão de teste: o alcance a partir de `cmd/` não se aplica, e conta-se pelas ocorrências em `*_test.go`.

| Padrão | Título | Estado na plataforma | Aplica | Evidência | Chamadores alcançáveis | Correção (`RFIX`) |
|---|---|---|---|---|---|---|
| [`cognee-revenu:PTN-004`](../../../cognee-revenu/.base/patterns/plataforma/PTN-004-versionamento-de-eventos.md) | Versionamento de eventos | ACEITE | **sim** | `internal/erasure/auditadapter.go:22`: `auditSchemaVersion  = "v1"` | 2 — `internal/erasure/auditadapter.go` | FIX-009 |
| [`cognee-revenu:PTN-006`](../../../cognee-revenu/.base/patterns/plataforma/PTN-006-outbox-transacional.md) | Outbox transacional | ACEITE | **sim** | `internal/erasure/auditadapter.go:95`: `INSERT INTO outbox_events (` | 1 — `internal/erasure/executor.go` | FIX-002, FIX-003 |
| [`cognee-revenu:PTN-016`](../../../cognee-revenu/.base/patterns/plataforma/PTN-016-ddd-tatico.md) | DDD tático (agregado, objeto de valor, evento de domínio, repositório) | ACEITE | **sim** | `internal/eventbus/publisher_adapters.go:15`: `func (p QuotePublisher) Publish(ctx context.Context, events []qdomain.DomainE…` | 14 — `internal/eventbus/eventbus.go` | FIX-023 |
| [`cognee-revenu:PTN-019`](../../../cognee-revenu/.base/patterns/plataforma/PTN-019-mensageria-kafka.md) | Mensageria Kafka — tópico, chave, grupo, produtor e offset | PROPOSTO | **parcial** | sem chamador alcançável a partir de cmd/ | — | FIX-001, FIX-009 |
| [`cognee-revenu:PTN-020`](../../../cognee-revenu/.base/patterns/plataforma/PTN-020-stack-geral.md) | Stack geral dos módulos Go e layout de módulo | PROPOSTO | **sim** | `go.mod`: go 1.26.10 | — | FIX-015, FIX-016, FIX-017, FIX-024 |
| [`cognee-revenu:PTN-023`](../../../cognee-revenu/.base/patterns/plataforma/PTN-023-portas-e-adaptadores.md) | Portas e adaptadores (arquitetura hexagonal) | PROPOSTO | **parcial** | `modules/admin/infrastructure/memory/repos.go` | — | FIX-015 |

**Sem sinal neste repositório:** `PTN-003`, `PTN-005`, `PTN-007`, `PTN-008`, `PTN-009`, `PTN-010`, `PTN-011`, `PTN-012`, `PTN-013`, `PTN-014`, `PTN-015`, `PTN-017`, `PTN-018`, `PTN-021`, `PTN-022`.

---

Gerado pela MS-018 6.3 da plataforma (`cognee-revenu`, O159 e O161) a partir da matriz medida da 6.2. Para acrescentar um padrão do módulo: copiar o `_template.md` do catálogo da plataforma, alocar o próximo número na tabela de cima e ligar o ADR que o adota (cognee-revenu:RN_CTX_026).

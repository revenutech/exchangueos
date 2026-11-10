---
data: 2026-11-10
tarefa: 1.2
estado: ☑️
evidencia: loader e cache da `archive_policy` (24 políticas; recarrega em UPDATE), provado contra o banco; N1 verde, na develop em e1f78f7; N2 do lote 9.2 pendente
pr:
---
Tarefa `RTSK.024.1002.01` (MS-024b 1.2), 2026-11-10, lote 9.2 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `internal/archive/policy.go`, testes de unidade e `policy_integration_test.go` (tag `integration`). `Interval` guarda meses, dias e microssegundos (1 ano = 12 meses, calendário por `AddDate`); `AgeCutoff` e `RetentionCutoff` (este devolve falso sob guarda legal). Impressão digital: contagem, `max(updated_at)` e md5 das demais colunas, consultada no máximo a cada 30 s; erro do banco é erro, nunca cópia velha.
- **teste com banco:** o teste de ida e volta roda os pacotes de `tests/ci-local/migracoes/go-integracao.txt` com o DSN do banco descartável e reprova teste pulado ou nenhum aprovado. Carrega 24 políticas; UPDATE com e sem `updated_at` chega ao cache.
- **achado:** o `escopo.conf` não lista as migrações 000010, 000011 e 000014 (linha ✏️ MS-024i 1.14).
- **N1:** pista com banco: «go integration: ./internal/archive/ (15 passed)»; cobertura de unidade 68,9% (o `PgxStore` só no teste com banco).
- **veredito do lote:** compara com a linha de base do accountos (p75 2,7 h). A do exchangeos ficou degenerada (`--medir-linha-de-base --desde 2026-11-01 --ate 2026-11-09`: 19 merges, 18 de registro; mediana 0,0 h, p75 0,0 h, p90 1,1 h). Decisão do dono por select na Core em 2026-11-10.

```prova
n1: f31594b4961fd6ce78846f0da9273e9c4045b98f rodada exchangeos-policy-loader@2026-11-10T22:32:56Z
n2: pendente — N2 única do lote 9.2, depois do último merge
integracao: develop e1f78f7e27bb48e66769cac8fde0cffe6abf42e5
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

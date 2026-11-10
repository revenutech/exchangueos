---
data: 2026-11-10
tarefa: 1.1
estado: ✅
evidencia: interface `SanctionsProvider` em `pkg/sanctions/`, com redação de PII em `String`/`LogValue`; revisada pela IdentityOS; N1 verde, na develop em ec66ab7; N2 do lote 9.1 verde em 88938b8
pr:
---
Tarefa `RTSK.024.1201.01` (MS-024d 1.1), 2026-11-10, lote 9.1 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `pkg/sanctions/provider.go`, `provider_test.go` e `redact_test.go`. `Refresh(ctx)`, `Match(ctx, MatchQuery) ([]Match, error)` e `Source()`; `MatchQuery` = Name, Country opcional e MinScore; `Score` é `float64` em [0, 1] (similaridade, não dinheiro).
- **revisão:** IdentityOS aprovou `37ed65f` com recomendação forte de redação; aplicada em `abe431b`: `String()` e `LogValue()` só com Country, MinScore e `name_len`; o teste com controle negativo nasceu vermelho (o JSONHandler imprimia o nome). Aprovado em `abe431b`.
- **débito (não bloqueia):** `%#v` usa `GoString()`, que falta, e ainda imprime o Name; `json.Marshal` direto do struct também. Vira tarefa de seguimento.
- **N1:** pista sem banco, build, vet, `test -race ./...` (34 pacotes) e `gitleaks` verdes.

```prova
n1: abe431ba0e898d01d05c41d5a316d59f78acad1f rodada exchangeos-sanctions@2026-11-10T15:07:31Z
n2: 88938b8ccca11c4f3200dc12c135a54a9c252167 rodada exchangeos-n2-lote91@2026-11-10T15:46:08Z — build, vet, test -race ./... (35 pacotes), govulncheck, gitleaks 663fdaf..head, escopo 8/8, lint-patterns e catraca head × base 663fdaf sem queda (33 → 35 pacotes)
integracao: develop ec66ab75896fcd90d05a216e9e04f6978ba0bff9
revisor: sessão IdentityOS — aprovado abe431b em 2026-11-10 (PII)
```

---
data: 2026-11-10
tarefa: 1.2
estado: ✅
evidencia: `scripts/lint-patterns.sh` com controle positivo e negativo; cada seção `FX-<FAM>-NNN` é um padrão; N1 verde, na develop em 3fc4239; N2 do lote 9.1 verde em 88938b8
pr:
---
Tarefa `RTSK.024.1702.01` (MS-024k 1.2), 2026-11-10, lote 9.1 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `scripts/lint-patterns.sh`, `tests/scripts/lint-patterns_test.sh` e as fixtures `completo.md` e `incompleto.md`.
- **unidade:** cada seção `##`/`###` `FX-<FAM>-NNN` de um catálogo é um padrão: 6 seções do `TEMPLATE.md` em ordem, ≥ 1 bloco de código, Anti-pattern não vazio e ≥ 1 link cruzado em Related. No catálogo 200, FX-GP-001 passa e FX-GP-002..005 falham, como esperado até as tarefas 1.4–1.10.
- **N1:** pista sem banco, `shellcheck`, o teste e `gitleaks` verdes.

```prova
n1: ae6d29c11b79612ba5b075e346df88c01ecaae84 rodada exchangeos-lint-patterns@2026-11-10T14:45:33Z
n2: 88938b8ccca11c4f3200dc12c135a54a9c252167 rodada exchangeos-n2-lote91@2026-11-10T15:46:08Z — build, vet, test -race ./... (35 pacotes), govulncheck, gitleaks 663fdaf..head, escopo 8/8, lint-patterns e catraca head × base 663fdaf sem queda (33 → 35 pacotes)
integracao: develop 3fc4239c599a4869414cb4202b1d5ec7ca35f4cd
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

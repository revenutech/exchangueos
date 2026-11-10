---
data: 2026-11-10
tarefa: 1.2
estado: ☑️
evidencia: `scripts/lint-patterns.sh` com controle positivo e negativo; cada seção `FX-<FAM>-NNN` é um padrão; N1 verde, na develop em 3fc4239; N2 do lote 9.1 pendente
pr:
---
Tarefa `RTSK.024.1702.01` (MS-024k 1.2), 2026-11-10, lote 9.1 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `scripts/lint-patterns.sh`, `tests/scripts/lint-patterns_test.sh` e as fixtures `completo.md` e `incompleto.md`.
- **unidade:** cada seção `##`/`###` `FX-<FAM>-NNN` de um catálogo é um padrão: 6 seções do `TEMPLATE.md` em ordem, ≥ 1 bloco de código, Anti-pattern não vazio e ≥ 1 link cruzado em Related. No catálogo 200, FX-GP-001 passa e FX-GP-002..005 falham, como esperado até as tarefas 1.4–1.10.
- **N1:** pista sem banco, `shellcheck`, o teste e `gitleaks` verdes.

```prova
n1: ae6d29c11b79612ba5b075e346df88c01ecaae84 rodada exchangeos-lint-patterns@2026-11-10T14:45:33Z
n2: pendente — N2 única do lote 9.1, depois do último merge
integracao: develop 3fc4239c599a4869414cb4202b1d5ec7ca35f4cd
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

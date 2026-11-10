---
data: 2026-11-10
tarefa: 1.11
estado: ☑️
evidencia: `.ci-local/escopo.conf` (6 categorias + 4 le-disco) e controle negativo 8/8; revisado pela Baremetal; N1 verde, na develop em 0fea661; N2 do lote 9.1 pendente
pr:
---
Tarefa `RTSK.024.1513.01` (MS-024i 1.11), 2026-11-10, lote 9.1 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `.ci-local/escopo.conf` no modelo do ledgeros (dinheiro, tenant, assinatura, pii, contrato, catalogo; 4 linhas `le-disco`) e `tests/ci-local/escopo-controle-negativo.sh` (um commit sintético por categoria, por plumbing, rodado num worktree destacado; `cmd/healthcheck` tem de dar «N1»).
- **revisão:** Baremetal pediu em `e7e8d19` as lacunas de dinheiro (cfets_*, cmd/eod, cmd/cls-cycle, quote/ e risk/ inteiros, pkg/bacen/, migrações 000002/3/4/6/7/8/12/13), tenant (modules/admin, pkg/outbox, 000009) e pii (000008), e no teste o `trap`, o FAIL claro para `first_file` vazio e o caso da 000002. Aprovado em `ee14ebe`.
- **N1:** pista sem banco, `shellcheck` limpo, controle 8/8, `--leituras` 4 declaradas e 0 não declaradas, `gitleaks` sem vazamento.

```prova
n1: ee14ebe2385096c25b68736c443d518396606a59 rodada exchangeos-escopo-conf@2026-11-10T15:40:20Z
n2: pendente — N2 única do lote 9.1, depois do último merge
integracao: develop 0fea661a2b44a4f5782bcf5407d83d484cc2aa12
revisor: sessão Baremetal — aprovado ee14ebe em 2026-11-10
```

---
data: 2026-11-10
tarefa: 1.3
estado: ✅
evidencia: migração 000014 (`screening_results.cos_case_id`, FK opcional por tenant); revisada pela IdentityOS; N1 verde, na develop em 9734733; N2 do lote 9.2 verde em cd359a1
pr:
---
Tarefa `RTSK.024.1403.01` (MS-024g 1.3), 2026-11-10, lote 9.2 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `migrations/000014_screening_cos_case_link.{up,down}.sql` e `tests/ci-local/migracoes/000014.{check,recusa}.sql`. FK `(tenant_id, cos_case_id)` com RESTRICT e índice parcial; recusa caso inexistente, caso de outro tenant e apagar caso ligado. O teste passa a descer uma migração por vez, rodando o down de novo logo depois do rollback dela.
- **ordem real:** o código veio antes do evento. Pela regra da DV (`dominio.py:61`), tarefa com risco só abre com a dependência entregue; a 1.3 abre depois da N2 do lote, com o `quando` real.
- **revisão:** IdentityOS aprovou `a562af9`; rebaseada sobre a 1.2 ajustada, reconfirmado em `2e42c68`.
- **N1:** pista com banco: recusas 23/23, «14 migrações, 17 checks».
- **veredito do lote:** compara com a linha de base do accountos (p75 2,7 h). A do exchangeos ficou degenerada (`--medir-linha-de-base --desde 2026-11-01 --ate 2026-11-09`: 19 merges, 18 de registro; mediana 0,0 h, p75 0,0 h, p90 1,1 h). Decisão do dono por select na Core em 2026-11-10.

```prova
n1: 2e42c6873db5cf6f4efd46f70ad00d78245e5359 rodada exchangeos-migracoes@2026-11-10T22:22:36Z
n2: cd359a19cba589a9996bc36afda70992309793cd rodada exchangeos-n2-lote92@2026-11-10T23:13:05Z — rodada-escopo n2 com 95 passos verdes (go test -race por pacote, build, vet, govulncheck, migracoes 14/17 checks com recusas e teste Go com banco, helm, escopo, passos, lint-patterns, shellcheck, gitleaks), gitleaks 78fa578..head e catraca head × base 78fa578 sem queda (35 → 36 pacotes)
integracao: develop 9734733c2a24b7b2e28276d88f8028a6024a585d
revisor: sessão IdentityOS — aprovado em a562af9, reconfirmado em 2e42c68 em 2026-11-10 (PII, tenant)
```

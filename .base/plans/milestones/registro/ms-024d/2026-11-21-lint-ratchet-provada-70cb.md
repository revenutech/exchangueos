---
data: 2026-11-21
tarefa: 1.17
estado: ✅
evidencia: N1 verde em 379b12f; develop c7167d2; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1219.01` (MS-024d 1.17), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** zera os 46 achados novos do golangci-lint (ratchet) sem mudar comportamento; 2 nolint com motivo.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** Lint 0 e testes verdes pela fila antes da N1; run.sh verde em 7d3f93c.
- **estado:** ✅ em 2026-11-22, conferido pelo dono (select «Passar as 14 a ✅»; a prova unitária basta, DV 2397).

```prova
n1: 379b12fcfe2f33b3b2daebe7dc81c01f1aa333e1 rodada exchangeos/MS-024d-1.17/n1 14:51 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop c7167d27ea9702ef035930243a04b7146072b0af
revisor: sessão IdentityOS — aprovado em 379b12f (PII; DV relato 2018)
```

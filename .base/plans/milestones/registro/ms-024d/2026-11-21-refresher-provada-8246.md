---
data: 2026-11-21
tarefa: 1.7
estado: ☑️
evidencia: N1 verde em d3978a2; develop 4b129a7; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1207.01` (MS-024d 1.7), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** `cmd/sanctions-refresher` grava um snapshot JSON por lista, de forma atômica.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **estado:** ☑️. O ✅ espera a conferência do dono (select de 2026-11-22, «Registros e ☑️»).

```prova
n1: d3978a266a6f2a0158fe90e6e5792beb773c177a rodada exchangeos/MS-024d-1.7/n1 13:55 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop 4b129a735eec495b49b03e3ba73a886feaa01de7
revisor: sessão IdentityOS — aprovado com 79cf1b5 e 7b2e9d7 (PII)
```

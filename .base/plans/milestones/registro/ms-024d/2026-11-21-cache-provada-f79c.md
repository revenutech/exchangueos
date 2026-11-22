---
data: 2026-11-21
tarefa: 1.2
estado: ☑️
evidencia: N1 verde em 928a385; develop 8a3b0cd; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1202.01` (MS-024d 1.2), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** cache em memória com índice invertido e pontuação Jaro.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** Já estava ☑️ no registro de 12/11 (cache-a3f1) com N2 pendente; este registro acrescenta a N2. O aceite de 10 ms só roda com `EXCHANGEOS_PERF=1` desde a 1.18.
- **estado:** ☑️. O ✅ espera a conferência do dono (select de 2026-11-22, «Registros e ☑️»).

```prova
n1: 928a3856166e1f6071bdd2a0662a5128fae68fec rodada exchangeos-n1@2026-11-12T07:40:29Z (rodada de 12/11) BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop 8a3b0cd71c0d58d426829e47f71cc23ab05d36c6
revisor: sessão IdentityOS — aprovado em 568e869 em 2026-11-11 (PII)
```

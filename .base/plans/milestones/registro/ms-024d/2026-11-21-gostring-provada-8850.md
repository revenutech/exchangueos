---
data: 2026-11-21
tarefa: 1.16
estado: ✅
evidencia: N1 verde em bf061e0; develop e2799dd; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1218.01` (MS-024d 1.16), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** `MatchQuery` redigido em `String`, `GoString`, `LogValue` e `MarshalJSON`.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** A 1ª N1 (1869a47, 14:07) reprovou: a constante de teste `syntheticName` colidia com a função do mesmo nome do cache (1.2). 257ea1b renomeou a constante para `redactMarkerName`, só no teste, depois da revisão.
- **estado:** ✅ em 2026-11-22, conferido pelo dono (select «Passar as 14 a ✅»; a prova unitária basta, DV 2397).

```prova
n1: bf061e09c7e50a8ab6a100d3e444d5f6ed72885c rodada exchangeos/MS-024d-1.16/n1 14:21 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop e2799dd5f8723383dc23aac1e67b11c7c2054d24
revisor: sessão IdentityOS — aprovado em 825d46c (PII)
```

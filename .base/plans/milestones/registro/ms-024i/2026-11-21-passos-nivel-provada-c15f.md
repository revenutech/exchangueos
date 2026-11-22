---
data: 2026-11-21
tarefa: 1.36
estado: ✅
evidencia: N1 verde em a931707; develop 0aa7ced; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1538.01` (MS-024i 1.36), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** o controle `passos-controle-negativo.sh` roda o `lista` sem herdar `RODADA_NIVEL`.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** Sob `RODADA_NIVEL=unitaria`: 1 falha sem o conserto, 6/6 com ele.
- **estado:** ✅ em 2026-11-22, conferido pelo dono (select «Passar as 14 a ✅»; a prova unitária basta, DV 2397).

```prova
n1: a9317072584c1797c42ae8b62e098b98350c2cbd rodada exchangeos/MS-024i-1.36/n1 14:15 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop 0aa7ced1706545ae8cae75833f426273f5dd17fd
revisor: não se aplica (CI local, sem dinheiro, tenant, assinatura ou PII)
```

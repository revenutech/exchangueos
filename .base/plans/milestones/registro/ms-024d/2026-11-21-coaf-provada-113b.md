---
data: 2026-11-21
tarefa: 1.6
estado: ☑️
evidencia: N1 verde em c236765; develop 3fbf51b; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1206.01` (MS-024d 1.6), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** leitor do XLSX do COAF só com a biblioteca padrão, limites contra zip bomb.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **estado:** ☑️. O ✅ espera a conferência do dono (select de 2026-11-22, «Registros e ☑️»).

```prova
n1: c2367658c846f229b2c91509b9f072f24ca36fcc rodada exchangeos/MS-024d-1.6/n1 13:55 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop 3fbf51b8c491938243a27ccca3c58138b351338d
revisor: sessão IdentityOS — aprovado em c061653 (PII)
```

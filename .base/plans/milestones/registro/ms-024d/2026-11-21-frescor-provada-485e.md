---
data: 2026-11-21
tarefa: 1.8
estado: ✅
evidencia: N1 verde em 321d935; develop d786765; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1208.01` (MS-024d 1.8), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** métrica `sanctions_list_age_seconds{list}` e regras de alerta de frescor.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** A 1ª N1 (c242ce6, 14:01) reprovou no controle `passos` por defeito da base, corrigido pela MS-024i 1.36; a N1 acima é a seguinte.
- **estado:** ✅ em 2026-11-22, conferido pelo dono (select «Passar as 14 a ✅»; a prova unitária basta, DV 2397).

```prova
n1: 321d9354b42caa585bbd2320e0f7da40371bfb00 rodada exchangeos/MS-024d-1.8/n1 14:19 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop d786765d9c37925464f84304cd78f5147d031f55
revisor: sessão IdentityOS — aprovado em ad0183b (PII)
```

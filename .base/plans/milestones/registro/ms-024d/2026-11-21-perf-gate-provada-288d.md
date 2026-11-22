---
data: 2026-11-21
tarefa: 1.18
estado: ☑️
evidencia: N1 verde em db16ee3; develop 837237f; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1220.01` (MS-024d 1.18), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** o aceite de 10 ms do cache (1.2) só roda com `EXCHANGEOS_PERF=1`.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** Notas da revisora, sem bloquear: medir o p99 < 10 ms com a máquina livre e registrar máquina e comando; a linha de perf com `EXCHANGEOS_PERF=1` entra no fechamento do plano (MS-024i 1.31); o pulo também tira da N2 o Load de 100k e os 1000 Match sem erro, que um `TestCache_Load100k` sem asserção de tempo devolveria.
- **estado:** ☑️. O ✅ espera a conferência do dono (select de 2026-11-22, «Registros e ☑️»).

```prova
n1: db16ee3862735dc7f5a43501af230f079703774f rodada exchangeos/MS-024d-1.18/n1 15:27 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop 837237f6cecedfd631a86a2cb6e6cd51439f2c64
revisor: sessão IdentityOS — aprovado em c490df2 (PII; 2026-11-22)
```

---
data: 2026-11-21
tarefa: 1.37
estado: ☑️
evidencia: N1 verde em 1e83bdf; develop 7d3f93c; N2 unitária verde em 837237f; na origin/develop
pr:
---
Tarefa `RTSK.024.1539.01` (MS-024i 1.37), provada em 2026-11-21 na série da ordem 1973 do DV (merge do que estava pronto; selects do dono «N1 e depois merge» e «run.sh e push»).

- **entrega:** golangci-lint com ratchet e o controle `golangci-novo-reprova.sh` como passos da N1.
- **prova:** N1 unitária (ADR-039) verde pela fila; merge na develop local; N2 unitária verde no head 837237f; origin/develop 837237f (push pelo hook, 2026-11-21 15:43 BRT, select do dono).
- **nota:** Os passos `golangci` e `golangci-controle` rodaram verdes na N1; R10 conferida pelo DV (ratchet_lint sem violação nova).
- **estado:** ☑️. O ✅ espera a conferência do dono (select de 2026-11-22, «Registros e ☑️»).

```prova
n1: 1e83bdf132cb7d3b429df87e95c10f4ddc691fe1 rodada exchangeos/MS-024i-1.37/n1 14:52 BRT 2026-11-21
n2: 837237f6cecedfd631a86a2cb6e6cd51439f2c64 rodada exchangeos/develop-837237f/n2 (unitária, ADR-039; catraca contra 530cea0) 2026-11-21 15:32 BRT
integracao: develop 7d3f93cfdc46de06e84cf223fe106660e00882b8
revisor: não se aplica (CI local, sem dinheiro, tenant, assinatura ou PII)
```

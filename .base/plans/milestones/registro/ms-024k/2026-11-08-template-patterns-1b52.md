---
data: 2026-11-08
tarefa: 1.1
estado: ☑️
evidencia: TEMPLATE.md em patterns/, N1 leve verde no sha de entrega
pr:
---
Tarefa `RTSK.024.1701.01` (MS-024k 1.1), 2026-11-08.

- **entrega:** `.base/plans/01-architecture/patterns/TEMPLATE.md` criado com as 6 seções exigidas (Context, Problem, Solution, Example, Anti-pattern, Related), ratchet ≤ ~400 palavras por seção e regras de forma (≥ 1 exemplo real, ≥ 1 anti-pattern, ≥ 1 cross-link no `Related`).
- **pattern de referência:** `FX-GP-001 — Aggregate constructor returns (*T, error)` em `.base/plans/01-architecture/patterns/200-fx-golang-patterns.md`, que já traz o template implícito e serviu de fonte da forma.
- **autorização:** select do dono no Core em 2026-11-08 09:20 BRT, extensão da etapa 10 do ajuste de granularidade ao exchangeos («até 2 tarefas abertas só de documentação ou plano, sem dinheiro, tenant, assinatura ou PII, com N1 leve local, sem banco, sem fila»).
- **☑️ provada (2026-11-08, 09:25 BRT):** N1 leve do § 6.10, régua do Core, verde no head de entrega `e491f0a8`:
  - `conferir-linhas.py` (do `baremetal/scripts/registro/`): **149 abertas, 0 falhas de formato** em todas as colunas (longas, sem-estado, sem-aceite, sem-entrega, sem-escopo, sem-espera, sem-link). Idêntico ao baseline pré-execução; `☑️` continua contando como aberta no `ESTADOS` do script, por desenho.
  - `gitleaks protect` e `gitleaks stdin` sobre o diff: `no leaks found`.
  - `base-lint`: ausente no exchangeos como ferramenta local (confirmado antes à Core: financeos/riskos/investos/accountos têm `tools/base-lint` com DDD hardcoded; nenhum porta genericamente, e o baremetal main 1.11.0 não tem um `base-lint` local em `scripts/`). Régua declarada inaplicável aqui — reportado no evento ao DV.
- **remoto:** push/PR/CD esperam o baremetal remoto. Fica tudo em `develop` local.

```prova
n1: e491f0a80e5d98a28bfa53a7d17c84f42f06cddf conferir-linhas+gitleaks 2026-11-08T12:25:30Z
n2: pendente
integracao: develop <sha-pós-merge>
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

---
data: 2026-11-12
tarefa: 1.12
estado: ☑️
evidencia: `ProviderScreener` com os 4 provedores por DI e falha fechada; N1, integração e N2 pendentes
pr:
---
Tarefa `RTSK.024.1212.01` (MS-024d 1.12), 2026-11-12, lote 9.4 da etapa 9 (composição aprovada pelo dono por select na Core em 2026-11-11).

- **entrega:** `ProviderScreener` com exatamente os 4 provedores por injeção de dependência; vence a maior pontuação; sem filtro de país, de propósito.
- **novo:** `CounterpartyNameResolver`; a ligação com o refdata ainda falta.
- **falha fechada:** `ErrScreeningFailed` com motivos fixos. A revisão da IdentityOS removeu o `%w` em volta dos erros do resolvedor e dos provedores, provado com um teste de nome sentinela.
- **em aberto:** `HitScope` não é definido em `ScreenCounterparty`; quando um auditor real for ligado, a triagem vai falhar com `ErrHitAudit` até a camada de serviço defini-lo.
- **ambiente:** as rodadas do N1 anteriores a 2026-11-11 15:42Z falharam só no govulncheck, em GO-2026-6859 (go1.26.10). O piso subiu para go1.26.11 na develop em 2a7bcf4 (MS-024i 1.15), e os ramos receberam a develop por merge, não por rebase, então os SHAs aprovados pela IdentityOS continuam valendo.
- **revisão:** IdentityOS, revisora de todas as tarefas do lote (`pkg/sanctions` segue `pii`, decisão do dono pela Core em 2026-11-11). Veredito: revisão a registrar (SHA aprovado a preencher).

```prova
n1: PENDENTE
n2: PENDENTE
integracao: PENDENTE
revisor: IndentityOS — revisão a registrar (SHA aprovado a preencher)
```

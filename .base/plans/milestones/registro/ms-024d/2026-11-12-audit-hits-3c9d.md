---
data: 2026-11-12
tarefa: 1.11
estado: ☑️
evidencia: um lote `SANCTIONS_HIT` por Match, com detalhe só acima do limiar; N1, integração e N2 pendentes
pr:
---
Tarefa `RTSK.024.1211.01` (MS-024d 1.11), 2026-11-12, lote 9.4 da etapa 9 (composição aprovada pelo dono por select na Core em 2026-11-11).

- **entrega:** um lote `SANCTIONS_HIT` por Match. Acertos no limiar ou acima levam {Source, EntryID, Score}; abaixo dele, só {Count, BestScore}, porque o detalhe é quase-identificador (IdentityOS).
- **falha fechada:** com auditor ligado, tenant vazio dá `ErrHitAudit`.
- **privacidade:** nenhum nome é gravado, nem em hash. A retenção está pendente com o DPO.
- **em aberto:** o adaptador que grava linhas em `audit_events` ainda não existe, e `correlation_id` é NOT NULL.
- **ambiente:** as rodadas do N1 anteriores a 2026-11-11 15:42Z falharam só no govulncheck, em GO-2026-6859 (go1.26.10). O piso subiu para go1.26.11 na develop em 2a7bcf4 (MS-024i 1.15), e os ramos receberam a develop por merge, não por rebase, então os SHAs aprovados pela IdentityOS continuam valendo.
- **revisão:** IdentityOS, revisora de todas as tarefas do lote (`pkg/sanctions` segue `pii`, decisão do dono pela Core em 2026-11-11). Veredito: revisão a registrar (SHA aprovado a preencher).

```prova
n1: PENDENTE
n2: PENDENTE
integracao: PENDENTE
revisor: IndentityOS — revisão a registrar (SHA aprovado a preencher)
```

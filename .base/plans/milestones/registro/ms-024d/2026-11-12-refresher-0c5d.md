---
data: 2026-11-12
tarefa: 1.7
estado: ☑️
evidencia: `cmd/sanctions-refresher` grava um snapshot JSON por lista, de forma atômica; N1, integração e N2 pendentes
pr:
---
Tarefa `RTSK.024.1207.01` (MS-024d 1.7), 2026-11-12, lote 9.3 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `cmd/sanctions-refresher` grava um snapshot JSON por lista, de forma atômica, modo 0600, com fsync do diretório. Os pods leem com `sanctions.ReadSnapshot`.
- **URLs:** vêm do ambiente, só https, e nunca aparecem em log nem em erro (a URL da UE carrega um token). TLS 1.3.
- **revisão:** a IdentityOS pediu que `CheckRedirect` recuse não-https e pare em 5 saltos (`79cf1b5`).
- **achado do N1:** host e porta vazavam por erros de dial e de DNS; corrigido com `stripAddr` (`7b2e9d7`). O nome do host no erro x509 foi julgado como não sendo dívida, porque os hosts das listas são públicos.
- **não verificado:** se os 4 hosts das listas aceitam TLS 1.3.
- **escopo:** o formato compartilhado fica em `pkg/sanctions/snapshot.go`, fora do escopo `cmd/` do plano, porque os pods precisam importá-lo.
- **decisões do dono do lote 9.3:** registradas uma vez, no registro da tarefa 1.2 (`2026-11-12-cache-a3f1.md`).
- **ambiente:** as rodadas do N1 anteriores a 2026-11-11 15:42Z falharam só no govulncheck, em GO-2026-6859 (go1.26.10). O piso subiu para go1.26.11 na develop em 2a7bcf4 (MS-024i 1.15), e os ramos receberam a develop por merge, não por rebase, então os SHAs aprovados pela IdentityOS continuam valendo.
- **revisão:** IdentityOS, revisora de todas as tarefas do lote (`pkg/sanctions` segue `pii`, decisão do dono pela Core em 2026-11-11). Veredito: aprovado com 79cf1b5 e 7b2e9d7.

```prova
n1: PENDENTE
n2: PENDENTE
integracao: PENDENTE
revisor: sessão IdentityOS — ajustar em cb13f20 (redirect), aprovado em 79cf1b5, reconfirmado em 7b2e9d7 (endereço do dial) em 2026-11-11 (PII)
```

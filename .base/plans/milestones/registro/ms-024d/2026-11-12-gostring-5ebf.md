---
data: 2026-11-12
tarefa: 1.16
estado: ☑️
evidencia: `MatchQuery` redigido em `String`, `GoString`, `LogValue` e `MarshalJSON`; N1, integração e N2 pendentes
pr:
---
Tarefa `RTSK.024.1218.01` (MS-024d 1.16), 2026-11-12, lote 9.4 da etapa 9 (composição aprovada pelo dono por select na Core em 2026-11-11).

- **entrega:** `String`, `GoString`, `LogValue` e `MarshalJSON` de `MatchQuery` passam todos por um único `redacted()`; o JSON leva só `country`, `min_score` e `name_len`. Controle negativo: `rawMatchQuery`. Fecha o débito do `%#v` registrado na tarefa 1.1.
- **nota de documentação:** um `MatchQuery` dentro de campo não exportado de struct formatada é impresso cru.
- **ambiente:** as rodadas do N1 anteriores a 2026-11-11 15:42Z falharam só no govulncheck, em GO-2026-6859 (go1.26.10). O piso subiu para go1.26.11 na develop em 2a7bcf4 (MS-024i 1.15), e os ramos receberam a develop por merge, não por rebase, então os SHAs aprovados pela IdentityOS continuam valendo.
- **revisão:** IdentityOS, revisora de todas as tarefas do lote (`pkg/sanctions` segue `pii`, decisão do dono pela Core em 2026-11-11). Veredito: aprovado em 825d46c; b15db04 só acrescenta ao comentário de PII que um MatchQuery em campo não exportado de struct formatada sai cru.

```prova
n1: PENDENTE
n2: PENDENTE
integracao: PENDENTE
revisor: sessão IdentityOS — aprovado em 825d46c, reconfirmado em b15db04 (só comentário) em 2026-11-11 (PII)
```

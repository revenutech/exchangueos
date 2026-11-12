# MS-024o — Publicador de taxa (`cambio.taxa_publicada`)

**Code:** `RMLS.024.020.01` M alocado pela Baremetal em 2026-11-12 (§ 3.3.5 da CONVENTIONS, baremetal dc38eae7, 1.16.12): M=20; 1ª tarefa `RTSK.024.2001.01`.
**Status:** BACKLOG
**Owner:** Pricing + Platform
**Origem:** decisão do dono por select na Core em 2026-11-12, a pedido da TreasuryOS (MS-016 2.3: o recálculo por taxa passa a ser disparado pela `cambio.taxa_publicada` do exchangeos).

## Por quê

A tesouraria (treasuryos, exposure) recalcula posição quando a taxa muda, e a ADR-006 dela (`RADR.283.001.01`) fixa que consome só `cambio.taxa_publicada`, por evento, nos quatro `tipo_taxa`, com a PTAX na mesma mensagem e um snapshot local com `rate_id`. Hoje o exchangeos não emite esse fato: não há publicador de taxa, nem o evento no código, nem a definição no catálogo de mensagens. A linha da TreasuryOS ficou ⏳ esperando isto.

## Descrição

Emitir `cambio.taxa_publicada` pelo outbox do exchangeos, com o identificador registrado no catálogo `ledgeros/.base/aasc/iso20022` e o envelope que o catálogo manda carimbar (`msg_def_idr`). O registro no catálogo foi feito pela sessão LedgerOS (decisão do dono, 2026-11-12): `xfxt.001.001.01`, classe C, tópico `revenu.xfxt.001`, emissor `exchangeos`, consumidor `treasuryos` (ledgeros 978c74db1, MS-079 1.9). Enums sem default: valor novo em tipo ou fonte vira `.002`. Frescor por `tipo_taxa` e RPC por `rate_id` seguem pendentes com o dono (ADR-006 da tesouraria) e ficam fora desta MS até decididos; a contingência `MANUAL` o dono decidiu em 2026-11-12 (linha 1.1).

## Fase 1 — Implementação

### Etapa 1.1 — Emissão do evento

- **1.1** `RTSK.024.2001.01` ✏️ Emitir `cambio.taxa_publicada` (`xfxt.001.001.01`, `MustID`) pelo outbox, uma cotação por mensagem; `MANUAL` só em contingência, com quem e motivo no audit. *Aceite:* `msgrepo events` sem E3; teste lê `revenu.xfxt.001` com `msg_def_idr` no header. *Entrega:* emissor + teste. *Escopo:* `modules/refdata/`, `pkg/outbox/`. *Espera:* sessão LedgerOS — revenu-common com o ID (lançamento pede o dono). [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.2002.01` ✏️ Rodar aceite da MS. *Aceite:* tarefas da Fase 1 cumpridas + N2 verde + a TreasuryOS confirma o recebimento do evento no consumidor de exposure. *Entrega:* relatório em `registro/ms-024o/`. *Escopo:* módulo inteiro. *Espera:* gate — rodada N2. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.2003.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- treasuryos ADR-006 `RADR.283.001.01` (fontes de taxa, frescor, contingência) e MS-016 2.3
- `.claude/rules/identificadores-de-mensagem.md` (identificador e envelope são contrato; ADR-060/061 do ledgeros)
- `pkg/outbox/kafka/publisher.go` (CloudEvents binário, referência da plataforma)
- `modules/refdata/infrastructure/olinda/ptax_fetcher.go` (fonte PTAX)

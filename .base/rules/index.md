# Catálogo de Regras — Índice

Regras `RRUL.DDD.FFF.VV` (alias `RN_<AREA>_NNN`). **O fundamento de uma
regra é um ADR (ou um padrão que ela obriga), nunca um fluxo**
(`RRUL.091.011.01`, cognee-revenu). Convenções em
[`CONVENTIONS.md`](CONVENTIONS.md); registro de domínios em
`baremetal/.base/plans/CONVENTIONS.md` § 3.3.5. Fluxos que as aplicam:
[`../flows/`](../flows/index.md). Versão: `VERSION`.

**Medido em 2026-10-22:** 11 regras do alias legado `RN_FX_*` migradas. **5
`ENFORCED` · 1 `AUDIT` · 5 `PROPOSED`.** Os detalhes de medição estão em
cada arquivo; abaixo é só o índice.

**O CLAUDE.md fala em 50 regras.** O número vem do SHACL do
`.base/aasc/ontology/compliance/bacen-cambio-shapes.ttl`. Medido no Go:
**11 aliases** aparecem no fonte, e deles **só 5** chegam a bloquear algo
no caminho de produção. As demais 39 são intenção de catálogo, não
controle — ficam fora deste índice até existirem.

| Alias | Code | Regra | Status | Arquivo |
|---|---|---|---|---|
| RN_FX_026 | `RRUL.024.001.01` | `decimal.Decimal` em *money* e *rate*; `float64` proibido | ENFORCED | [gov.md](gov.md) |
| RN_FX_001 | `RRUL.221.001.01` | Par de moedas: ambas ISO 4217 alfa-3 e distintas | ENFORCED | [trade.md](trade.md) |
| RN_FX_002 | `RRUL.221.002.01` | *Trade type* dentro do vocabulário (SPOT, FORWARD, SWAP, NDF); SPOT = T+2 por padrão | ENFORCED | [trade.md](trade.md) |
| RN_FX_005 | `RRUL.221.003.01` | NDF só em par com *fixing source* configurado | PROPOSED | [trade.md](trade.md) |
| RN_FX_013 | `RRUL.221.004.01` | *Amendment* acima de USD 100k exige revisão de quatro olhos | PROPOSED | [trade.md](trade.md) |
| RN_FX_010 | `RRUL.223.001.01` | PvP via CLS para os 18 pares elegíveis | AUDIT | [settle.md](settle.md) |
| RN_FX_017 | `RRUL.226.001.01` | SSI obrigatória antes da primeira liquidação com a contraparte | PROPOSED | [refdata.md](refdata.md) |
| RN_FX_015 | `RRUL.227.001.01` | NOP monitorada em tempo real; *halt* se exceder o teto do BCB | PROPOSED | [risk.md](risk.md) |
| RN_FX_028 | `RRUL.228.001.01` | Código de natureza válido (95 códigos da Circ. BCB 3.690) | ENFORCED | [comp.md](comp.md) |
| RN_FX_037 | `RRUL.228.002.01` | IOF cambial automático (6 alíquotas Decreto 12.499/2025) | ENFORCED | [comp.md](comp.md) |
| RN_FX_039 | `RRUL.228.003.01` | COS para SISCOAF em até 1 dia útil após a decisão de risco | PROPOSED | [comp.md](comp.md) |

**Próximos:** `RN_GOV_002` / `RRUL.024.002` · `RN_TRADE_005` / `RRUL.221.005` ·
`RN_QUOTE_001` / `RRUL.222.001` · `RN_SETTLE_002` / `RRUL.223.002` ·
`RN_CFETS_001` / `RRUL.224.001` · `RN_SETT2_001` / `RRUL.225.001` ·
`RN_REFDATA_002` / `RRUL.226.002` · `RN_RISK_002` / `RRUL.227.002` ·
`RN_COMP_004` / `RRUL.228.004` · `RN_EOD_001` / `RRUL.229.001`.

# RN_TRADE — Agregado `FXTrade` e *amendment*

Regras do agregado `FXTrade` (`modules/trade/`) e de *amendment*
(`modules/amendment/`).

### RN_FX_001 — Par de moedas: ambas ISO 4217 alfa-3 e distintas

| Campo | Valor |
|---|---|
| **Code** | `RRUL.221.001.01` · alias `RN_FX_001` |
| **Status** | ENFORCED |
| **Origem** | MS-023 |
| **Regra** | Um *trade* é recusado se `bought_ccy` ou `sold_ccy` não forem código ISO 4217 alfa-3, ou se forem iguais, ou se `buyer_bic == seller_bic`. |
| **Racional** | Par mal-formado ou *self-trade* não é FX. |
| **Imposta por** | `modules/trade/domain/input.go:36–72` (validações `tenant_id required`, `trade_type`, `venue`, `buyer_bic`, `seller_bic`, `buyer and seller cannot be the same party`, `bought_amount must be > 0`, `sold_amount must be > 0`, `deal_rate must be > 0`, `trade_date required`) e `fxtrade_test.go:54,63` (testes `TestRN_FX_001_CurrencyPairMustDiffer` e `TestRN_FX_001_CurrencyMustBeISO4217Alpha3`). |
| **Evidência** | `go test ./modules/trade/domain/ -run RN_FX_001` |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.001.01 (Book FX Spot via CLS), RFLW.024.002.01 |
| **Desde** | 2026-01 (MS-023) |

### RN_FX_002 — *Trade type* no vocabulário; SPOT = T+2 por padrão

| Campo | Valor |
|---|---|
| **Code** | `RRUL.221.002.01` · alias `RN_FX_002` |
| **Status** | ENFORCED |
| **Origem** | MS-023 |
| **Regra** | `trade_type` só aceita `SPOT`, `FORWARD`, `SWAP` ou `NDF`. Default de `settlement_date` em SPOT é T+2 (T+1 para USD/CAD e USD/MXN). |
| **Racional** | *Trade type* fora do vocabulário é dado inválido. T+2 é o padrão de mercado e evita que uma omissão vire data arbitrária. |
| **Imposta por** | `modules/trade/domain/fxtrade.go:12` (`TradeType` enum); `input.go:39` recusa `trade_type` fora do enum. |
| **Evidência** | `go test ./modules/trade/domain/` |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.001.01 |
| **Desde** | 2026-01 (MS-023) |

### RN_FX_005 — NDF só em par com *fixing source* configurado

| Campo | Valor |
|---|---|
| **Code** | `RRUL.221.003.01` · alias `RN_FX_005` |
| **Status** | PROPOSED — o código cita a lista de moedas elegíveis em `pkg/pricing/ndf.go:13` como comentário; nenhum caminho de produção recusa NDF por falta de *fixing source*. O `BookFXTrade` não consulta o *refdata* de *fixing*. |
| **Origem** | MS-023 |
| **Regra** | *Trade* `NDF` é recusado se o par não tiver *fixing source* (PTAX/WMR/ECB) e *fixing date* configurados em *refdata*. |
| **Racional** | NDF sem *fixing* não liquida — vira pendência. |
| **Imposta por** | — (alvo: `modules/refdata/domain/currency_pair.go` + chamada no `BookFXTrade`) |
| **Evidência** | — |
| **Exceções** | nenhuma |
| **Fluxos** | — |
| **Desde** | 2026-01 (MS-023) |

### RN_FX_013 — *Amendment* acima de USD 100k exige quatro-olhos

| Campo | Valor |
|---|---|
| **Code** | `RRUL.221.004.01` · alias `RN_FX_013` |
| **Status** | PROPOSED — citada em `pkg/iso20022/fxtr/fxtr_014_001_05.go:30` e `fxtr_015_001_05.go:30` como «Optional audit / 4-eyes block». O comentário diz «mandatory above USD 100k» mas nenhum caminho de produção verifica um segundo aprovador. |
| **Origem** | MS-023 |
| **Regra** | *Amendment* com valor acima de USD 100k é recusado sem registro de segundo aprovador. |
| **Racional** | Risco de erro operacional e de fraude cresce com valor; quatro olhos reduz ambos. |
| **Imposta por** | — (alvo: `modules/amendment/application/command/` + política de limite) |
| **Evidência** | — |
| **Exceções** | nenhuma |
| **Fluxos** | — |
| **Desde** | 2026-01 (MS-023) |

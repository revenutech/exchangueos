# RN_SETTLE — Liquidação CLS

Regras do subdomínio de liquidação CLS: PvP, PayIn, NetReport, `admi`
(`modules/cls_settlement/`, `payin/`, `netreport/`, `admin/`).

### RN_FX_010 — PvP via CLS para os 18 pares elegíveis

| Campo | Valor |
|---|---|
| **Code** | `RRUL.223.001.01` · alias `RN_FX_010` |
| **Status** | AUDIT — o roteador ISO 20022 escolhe a variante `fxtr` CLS quando o `settlement_venue == CLS` (`pkg/iso20022/registry/router.go:10`), e o aggregate `FXTrade` carrega `SettlementVenue` como enum. O que ainda **não** é imposto: recusar `settlement_venue != CLS` para um par CLS-elegível. Em produção um *trade* num par elegível pode ir para *gross* sem alerta. |
| **Origem** | MS-023 |
| **Regra** | *Trade* num dos 18 pares CLS-elegíveis é liquidado via CLS PvP, salvo exceção registrada. |
| **Racional** | PvP elimina risco Herstatt; CLS é o único trilho PvP para FX à vista. |
| **Imposta por** | `pkg/iso20022/registry/router.go` (seleção de variante por `venue`); `modules/trade/domain/fxtrade.go:34` (`SettlementVenue`). Não há recusa quando o par é elegível e o `venue` escolhido não é CLS. |
| **Evidência** | `go test ./pkg/iso20022/registry/` (seleção de variante); falta teste de recusa. |
| **Exceções** | em revisão — a lista de 18 pares vive em `modules/refdata/`. |
| **Fluxos** | RFLW.024.001.01 (Book FX Spot via CLS), RFLW.024.020.01 |
| **Desde** | 2026-01 (MS-023) |

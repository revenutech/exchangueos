# RN_RISK — Risco e posição

Regras do subdomínio de risco e posição (`modules/risk/`, `modules/position/`).

### RN_FX_015 — NOP monitorada em tempo real; *halt* se exceder teto do BCB

| Campo | Valor |
|---|---|
| **Code** | `RRUL.227.001.01` · alias `RN_FX_015` |
| **Status** | PROPOSED — o próprio código declara **que a regra NÃO estava sendo imposta** por esta RPC: «limit_type and scope used to be hardcoded to LimitCounterparty and the trade_id. A trade id is not a limit scope — limits are configured per counterparty BIC, currency or tenor — so repo.Find never matched and the check returned NotFound for every call. RN_FX_015 was not enforced through this RPC at all.» (`modules/risk/api/grpc_server.go:39–43`). A correção foi parcial: hoje o `scope` chega do chamador, mas nenhum caminho **publica** o limite por BIC/moeda/tenor, nem **halt** a aceitação de *trade* por NOP excedida — a RPC apenas devolve `Allowed: true/false` a quem perguntar. |
| **Origem** | MS-023 (declarada) · correção parcial em `grpc_server.go` |
| **Regra** | Posição Líquida Aberta (NOP) é consultada antes de aceitar um *trade*; se o *trade* fizer a NOP exceder o teto configurado por banco/moeda, o *trade* é recusado. |
| **Racional** | NOP excedida é descumprimento regulatório (Lei 14.286, Res BCB) e exposição que vira perda em movimento adverso de mercado. |
| **Imposta por** | — (alvo: `modules/trade/application/command/BookFXTrade` consulta `risk.CheckLimit` por BIC/moeda/tenor **antes** da transição `Captured → Confirmed`; recusa com erro de domínio no caminho de aceitação, não só em RPC consultiva) |
| **Evidência** | — |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.060.01 (risk check) |
| **Desde** | 2026-01 (MS-023) |

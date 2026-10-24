# RN_REFDATA — `CurrencyPair`, calendars, SSI, `Counterparty`

Regras de dados de referência (`modules/refdata/`).

### RN_FX_017 — SSI obrigatória antes da primeira liquidação

| Campo | Valor |
|---|---|
| **Code** | `RRUL.226.001.01` · alias `RN_FX_017` |
| **Status** | PROPOSED — o aggregate `SSI` existe em `modules/refdata/domain/ssi.go` e o comentário cita a regra; nenhum caminho de produção recusa a primeira liquidação pela ausência de SSI. O `cls_settlement` não consulta `SSI` antes do `fxtr.008` CLS. |
| **Origem** | MS-023 |
| **Regra** | A primeira liquidação com uma contraparte é recusada se não houver SSI (*Standard Settlement Instruction*) registrada no par (moeda, contraparte). |
| **Racional** | Liquidar sem SSI é dinheiro para endereço desconhecido — vira *fail* e, no limite, perda. |
| **Imposta por** | — (alvo: `modules/cls_settlement/application/command/` consulta `refdata.SSIRepository` antes de emitir `fxtr.008`) |
| **Evidência** | — |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.020.01 (CLS trade capture) |
| **Desde** | 2026-01 (MS-023) |

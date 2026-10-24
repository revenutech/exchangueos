# RN_COMP — *Compliance* BACEN

Regras do subdomínio de *compliance* (`modules/compliance/`, `pkg/bacen/`):
DEC, SCE, IOF, SISCOAF, sanções.

### RN_FX_028 — Código de natureza válido (95 códigos Circ. BCB 3.690)

| Campo | Valor |
|---|---|
| **Code** | `RRUL.228.001.01` · alias `RN_FX_028` |
| **Status** | ENFORCED |
| **Origem** | MS-023 · MS-024e |
| **Regra** | A classificação de operação é recusada se o código não for um dos 95 códigos da Circular BCB 3.690, ou se o texto livre não for mapeado para nenhum. |
| **Racional** | Classificação errada vira auto-de-infração do BCB. Lei 14.286, Res. BCB 277-561. |
| **Imposta por** | `modules/compliance/application/service.go:105–130` (`ClassifyOperation`) chama `bacen.Classifier.ByCode` e, no fallback, `Classify(hint)`; o `NewClassification` recusa `code 4-6 digits` fora do padrão (`classification.go:46–60`) e `nature` fora de `REMESSA|INGRESSO|CONVERSAO`. |
| **Evidência** | `go test ./modules/compliance/...` · `pkg/bacen/classifier_test.go` |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.030.01 (BACEN classification) |
| **Desde** | 2026-01 (MS-023); correção parcial no commit `526284b` |

### RN_FX_037 — IOF cambial automático (6 alíquotas Decreto 12.499/2025)

| Campo | Valor |
|---|---|
| **Code** | `RRUL.228.002.01` · alias `RN_FX_037` |
| **Status** | ENFORCED — `bacen.IOFCalculator.Compute` resolve alíquota por `opType` e aplica sobre `notional`; `ComputeIOF` persiste. Operação com `opType` fora das 6 categorias conhecidas é recusada no `Compute`. **Nota de medição:** só é imposto onde o *caller* chama `ComputeIOF` — e o `BookFXTrade` não liga ao serviço de IOF no caminho atual; o `cmd/worker` consome o evento. Como regra do SUBDOMÍNIO de *compliance* é ENFORCED; como *obrigação de pipeline* é AUDIT, e a diferença fica explícita aqui. |
| **Origem** | MS-023; alíquotas atualizadas pelo Decreto 12.499/2025 |
| **Regra** | O IOF é calculado automaticamente para toda operação cambial, pela alíquota do tipo de operação (6 categorias) multiplicada pelo nocional. |
| **Racional** | IOF subdeclarado é descumprimento tributário. |
| **Imposta por** | `modules/compliance/application/service.go:132–152` (`ComputeIOF`) · `pkg/bacen/iof.go` · `modules/compliance/domain/iof.go` |
| **Evidência** | `go test ./pkg/bacen/... ./modules/compliance/...` |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.030.01 |
| **Desde** | 2026-01 (MS-023) |

### RN_FX_039 — COS para SISCOAF em até 1 dia útil após decisão de risco

| Campo | Valor |
|---|---|
| **Code** | `RRUL.228.003.01` · alias `RN_FX_039` |
| **Status** | PROPOSED — a triagem de sanções hoje é *fail-closed* no `screener nil` (`service.go:88–91` injeta `UnavailableScreener`), e o `ScreeningResult` deriva `risk_level` dos *hits*. O que ainda **não existe**: emissão de COS para a SISCOAF quando o `risk_level == HIGH`. O comentário em `service.go:56` diz: «under RN_FX_039 … clear at RiskLow without a single watchlist being consulted» — a correção do commit `d4865ad` fechou o *fail-open*; a emissão do COS continua ausente. |
| **Origem** | MS-023 · MS-024g (SISCOAF submission) |
| **Regra** | Toda triagem com `risk_level == HIGH` dispara uma Comunicação de Operação Suspeita (COS) à SISCOAF em até 1 dia útil. |
| **Racional** | PLD/FT. Lei 9.613/1998 e normativos BCB; penalidades previstas em Res. BCB. |
| **Imposta por** | — (alvo: *saga* reativa a `ScreeningResult.risk_level == HIGH` → `cmd/worker` emite XML COS para SISCOAF) |
| **Evidência** | — |
| **Exceções** | nenhuma |
| **Fluxos** | RFLW.024.030.01 (hoje descreve só a triagem) |
| **Desde** | 2026-01 (MS-023) |

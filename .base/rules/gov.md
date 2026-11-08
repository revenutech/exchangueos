# RN_GOV — Governança transversal do módulo

Regras que atravessam todos os subdomínios do `exchangeos`: representação
de *money* e *rate*, caixa de ferramentas comum, segredos, segurança.

### RN_FX_026 — `decimal.Decimal` em *money* e *rate*; `float64` proibido

| Campo | Valor |
|---|---|
| **Code** | `RRUL.024.001.01` · alias `RN_FX_026` |
| **Status** | ENFORCED |
| **Origem** | MS-023 (fundação FX-FND-001); `pkg/pricing` |
| **Regra** | Todo valor monetário e toda taxa em `money` e `rate` usa `github.com/shopspring/decimal`. `float64` em assinaturas de função ou campos de agregado é proibido. |
| **Racional** | *Float* propaga erro de arredondamento e vira perda contábil silenciosa. A base NUMERIC(20,8) do CRDB exige representação decimal exata. |
| **Imposta por** | assinatura dos tipos `Money` e `Rate`: `modules/trade/domain/input.go`, `pkg/pricing/*.go` só expõem `decimal.Decimal` (validações `must be > 0`, `cannot be negative`, `must be a fraction (<= 1)` em `input.go:41–47`); o compilador recusa chamadas com `float64` por tipagem. |
| **Evidência** | `go vet ./... && go test ./pkg/pricing/...`; busca `grep -r 'float64' modules/ pkg/pricing/` retorna só código de teste e serialização JSON. |
| **Exceções** | nenhuma |
| **Fluxos** | aplicável a todos |
| **Desde** | 2026-01 (MS-023) |

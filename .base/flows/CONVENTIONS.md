# Catálogo de Fluxos — Convenções

Version: 1.0.0 | Atualizado: 2026-10-22 | Espelha `baremetal/.base/flows/CONVENTIONS.md`,
`ledgeros/.base/flows/CONVENTIONS.md` e `treasuryos/.base/flows/CONVENTIONS.md`:
mesmo código `RFLW`, registro de domínios disjunto (**`024` + `220`–`249` =
exchangeos**, `baremetal/.base/plans/CONVENTIONS.md` § 3.3.5).

**Caminho canônico:** todo fluxo do exchangeos vive **apenas** em
`.base/flows/`. ADRs, regras e documentação **referenciam** o código do
fluxo; não redesenham o fluxo em outro lugar.

**Fluxo descreve, não decide nem obriga** (`RULE-008` do `onboardos`). Ele
descreve o que o código faz **hoje**. Quando o código faz algo diferente do
esperado, o fluxo registra isso como **achado**, com arquivo e linha; não
descreve o comportamento desejado como se fosse o atual.

## 1. Código — `RFLW.DDD.FFF.VV`

```
RFLW.DDD.FFF.VV
  |    |   |   +-- VV  = versão do fluxo (01, 02, …) — incrementa em reescrita substancial
  |    |   +------ FFF = sequencial dentro do subdomínio (001–999)
  |    +---------- DDD = subdomínio do exchangeos (`024` legado + `220`–`249`)
  +--------------- RFLW = prefixo fixo (Revenu FLow)
```

### 1.1 Subdomínios — recorte de `baremetal/.base/plans/CONVENTIONS.md` § 3.3.5

| DDD | Diretório | Subdomínio |
|-----|-----------|------------|
| 024 | `legacy/` (os 8 fluxos existentes nas suas pastas atuais) | governança do módulo + legado; nenhum fluxo novo nasce aqui, só os 8 pré-existentes ficam |
| 221 | `trade/` | agregado `FXTrade`, *amendment*, *cancellation*, *novation* |
| 222 | `quote/` | RFQ e `FXQuoteRequest` |
| 223 | `cls_settlement/` | liquidação CLS: PvP, PayIn, NetReport, `admi` |
| 224 | `cfets/` | CFETS *Trade Capture* e *Confirmation* |
| 225 | `settlement/` | liquidação não-CLS: *gross*, BACEN Tx 70 |
| 226 | `refdata/` | `CurrencyPair`, calendars, SSI, `Counterparty` |
| 227 | `risk/` | `TradingLimit`, NOP, VaR, MTM, P&L |
| 228 | `compliance/` | DEC, SCE, IOF, SISCOAF, sanções |
| 229 | `eod/` | *batch* fim de dia, PTAX, revaluation |

### 1.2 Regras do catálogo

- **DDD imutável**; **FFF sequencial** (próximo número alocado em
  `index.md`); **VV** começa em 01.
- Fluxo descontinuado mantém o código para sempre (`Status: DEPRECATED`);
  códigos nunca são reutilizados.
- Todo fluxo lista as **regras `RN_*`** que aplica (canônicas em
  [`../rules/`](../rules/index.md)).
- Todo fluxo tem `Traceability`: código, teste, *runbook* ou *workflow*
  que o materializam.
- **`ACTIVE` exige execução real num ambiente.** Fluxo implementado no
  código e nunca executado fora de teste é `DRAFT`. Em 2026-10-22, nenhum
  fluxo do `exchangeos` está `ACTIVE` — não há ambiente operando.

## 2. Template

Dois formatos convivem no catálogo durante a migração:

**Formato antigo (YAML *front matter* — os 8 fluxos legados `RFLW.024.*`
mantêm esse formato até reescrita substancial):**

```markdown
---
Code: RFLW.024.NNN.NN
Domain: 024 (ExchangeOS)
...
---
```

**Formato novo (tabela — todo fluxo novo em `221`–`249` nasce assim):**

```markdown
# RFLW.DDD.FFF.VV — Nome do fluxo

| Campo | Valor |
|-------|-------|
| **Code** | RFLW.DDD.FFF.VV |
| **Domain** | nome |
| **Status** | DRAFT / ACTIVE / DEPRECATED |
| **Created** / **Updated** | AAAA-MM-DD |
| **Owner** | papel |
| **Traceability** | código, teste |
| **Rules** | RN_XXX_001, … |
| **ADRs** | ADR-NNN |

## Descrição
## Gatilho e conclusão
## Pré-condições
## Fluxo principal (sequenceDiagram)
## Fluxos alternativos / erro
## Achados
## Regras aplicadas
| Código | Regra | Onde é imposta |
```

## 3. Versionamento

Semver em `VERSION`. Subdomínio novo ou mudança estrutural = MAJOR; fluxo
novo = MINOR; correção = PATCH. O PR não faz bump nem escreve no
`CHANGELOG.md`: deixa um fragmento em `changelog.d/`
(`AAAA-MM-DD-<slug>.md`), e a versão só muda num release do catálogo. O
`index.md` é editado no próprio PR, com uma linha por fluxo.

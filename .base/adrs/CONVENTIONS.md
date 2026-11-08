# Catálogo de ADRs — Convenções

Version: 1.0.0 | Atualizado: 2026-10-22 | Espelha `baremetal/.base/adrs/` e segue
`onboardos-revenu/.base/adrs/README.md` § «O que exige uma ADR».

**Caminho canônico:** toda decisão arquitetural do exchangeos vive **apenas** em
`.base/adrs/`. README, planos e documentação **referenciam** o código da ADR;
não redecidem em outro lugar (`RULE-008`: doc explica, não decide).

## 1. Código — `RADR.DDD.FFF.VV` + alias `ADR-NNN`

**Código canônico** no padrão unificado (`baremetal/.base/plans/CONVENTIONS.md`
§ 3, faixa do exchangeos em § 3.3.5): `RADR` + subdomínio + sequência por
subdomínio + versão. **Alias humano** `ADR-NNN`, sequencial neste catálogo. O
número (`ADR-NNN`) e o `FFF` são **alocados no `index.md`, no mesmo commit do
documento**. Não existem números reservados.

| `DDD` | Subdomínio | Quando a ADR vai para cá |
|---|---|---|
| 024 | governança do módulo + legado | decisão transversal ao módulo: catálogos, contratos, dependências, segredos, build; ADRs transversais nascem aqui |
| 221 | *trade* | decisões específicas do agregado `FXTrade` e *amendment* |
| 222 | *quote* | RFQ, matching, cotação |
| 223 | *cls_settlement* | PvP CLS, PayIn, NetReport |
| 224 | CFETS | capture + confirmation |
| 225 | *settlement* | não-CLS, Tx 70, *gross* |
| 226 | *refdata* | pares, calendars, SSI, counterparty |
| 227 | risco / posição | NOP, VaR, MTM |
| 228 | *compliance* BACEN | DEC, SCE, IOF, SISCOAF |
| 229 | fim de dia | PTAX, revaluation, snapshots |

## 2. Status

Vocabulário fechado: `DRAFT` | `PROPOSED` | `ACCEPTED` | `DEPRECATED` |
`SUPERSEDED` | `REJECTED`. Forma única no corpo: `**Status:** X` no início
de linha, valor em MAIÚSCULAS, qualificação opcional depois (ex.: `**Status:**
PROPOSED — aguardando medição de banda em homologação`).

**Aceito é imutável.** Para mudar o rumo, nasce uma ADR nova que **supersede**
a anterior, e as duas atualizam o status no mesmo commit.

## 3. O que exige uma ADR

Uma decisão vira ADR quando passa **nas quatro** perguntas (herdado de
`onboardos-revenu/.base/adrs/README.md`):

1. **Alternativa real:** existia opção plausível que foi descartada por motivo?
2. **Porta cara de reabrir:** mudar a decisão depois obriga a tocar código,
   migrar dado ou renegociar contrato?
3. **Atravessa contextos:** mais de um BC/time/sistema sente a consequência?
4. **Porquê não dedutível do código:** o fonte sozinho não explica a razão?

Decisão que **não** passa fica listada no `index.md` como «decisão de doc,
sem ADR», com o motivo — não some, mas também não vira ADR.

## 4. Versionamento e índice

`VERSION` + `CHANGELOG.md` (semver: campo novo no template = MAJOR; ADR nova =
MINOR; correção = PATCH). O PR não faz bump nem escreve no `CHANGELOG.md`:
deixa um fragmento em `changelog.d/` (`AAAA-MM-DD-<slug>.md`), e a versão só
muda num release do catálogo. O `index.md` lista todas as ADRs com `ADR-NNN`,
código `RADR.DDD.FFF.VV` e status; o próximo `ADR-NNN` **e** o próximo `FFF`
por subdomínio são alocados lá.

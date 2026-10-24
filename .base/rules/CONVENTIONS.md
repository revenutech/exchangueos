# Catálogo de Regras — Convenções

Version: 1.0.0 | Atualizado: 2026-10-22 | Espelha `baremetal/.base/rules/CONVENTIONS.md`
e `treasuryos/.base/rules/CONVENTIONS.md`: **caminho canônico único, código
imutável alocado no índice, status com vocabulário fechado**.

**Caminho canônico:** toda regra do exchangeos é declarada **apenas** em
`.base/rules/`. O código que a impõe (especificação, agregado, política, SHACL
*shape*, migração) **cita o alias** `RN_FX_*` em comentário ou mensagem de
erro. Uma regra sem artefato que a imponha **no caminho de produção** é
`PROPOSED`, não `ENFORCED`.

**O Claude Code não carrega esta pasta sozinho.** Só `.claude/rules/` chega a
toda sessão. Por isso o `CLAUDE.md` passa a apontar para cá. A regra de
plataforma `.claude/rules/identificadores-de-mensagem.md` continua lá, porque
é cópia da regra canônica da plataforma e não é deste catálogo.

## 1. Código — `RRUL.DDD.FFF.VV` + alias `RN_<AREA>_NNN`

**Código canônico** no padrão unificado (`baremetal/.base/plans/CONVENTIONS.md`
§ 3, faixa do exchangeos em § 3.3.5): `RRUL` + subdomínio + sequência por
subdomínio + versão. **Alias humano** `RN_<AREA>_NNN`, com área = arquivo
onde a regra vive. Os aliases **já existiam no código** desde o MS-023 e
foram mantidos sem renumerar — por isso vários `NNN` são «altos» (017, 026,
037, 039): o código do catálogo é novo, o alias não muda. Alias e código
são imutáveis e nunca reutilizados; o `index.md` resolve um no outro.

| AREA | Arquivo | `DDD` | Escopo |
|------|---------|-------|--------|
| `GOV` | [gov.md](gov.md) | 024 | governança transversal: *money*/*rate*, outbox, idempotência, segredos. **Também recebe as regras legadas** que estão em código mas atravessam todo o módulo |
| `TRADE` | [trade.md](trade.md) | 221 | agregado `FXTrade`, *amendment*, *cancellation*, *novation* |
| `QUOTE` | [quote.md](quote.md) | 222 | RFQ e `FXQuoteRequest` |
| `SETTLE` | [settle.md](settle.md) | 223 | liquidação CLS: PvP, PayIn, NetReport, `admi` |
| `CFETS` | [cfets.md](cfets.md) | 224 | CFETS: *Trade Capture* e *Confirmation* |
| `SETT2` | [sett2.md](sett2.md) | 225 | liquidação não-CLS: *gross*, BACEN Tx 70 |
| `REFDATA` | [refdata.md](refdata.md) | 226 | `CurrencyPair`, calendars, SSI, `Counterparty` |
| `RISK` | [risk.md](risk.md) | 227 | `TradingLimit`, NOP, VaR, MTM, P&L |
| `COMP` | [comp.md](comp.md) | 228 | *compliance* BACEN: DEC, SCE, IOF, SISCOAF, sanções |
| `EOD` | [eod.md](eod.md) | 229 | *batch* fim de dia, PTAX, revaluation |

**Nota sobre o alias legado `RN_FX_*`.** O `CLAUDE.md` do exchangeos cita «50
regras `RN_FX_001..050`», número que vem da planilha de SHACL *shapes* e dos
comentários do SHACL do `.base/aasc/ontology/compliance/`. **Medido em
2026-10-22: só 11 `RN_FX_*` aparecem no código Go** — as outras 39 são
intenção de catálogo, não controle. Mantém-se o alias `RN_FX_NNN` para
quem já está no código (RN_FX_001, 002, 005, 010, 013, 015, 017, 026, 028,
037, 039), com a área real da matéria (ex.: `RN_FX_015` fica em `RISK`; o
alias não muda). Regras novas usam `RN_<AREA>_NNN` da área direta
(`RN_TRADE_001`, `RN_SETTLE_001`, …), não `RN_FX_*`.

## 2. Status

`PROPOSED` (declarada, sem artefato que bloqueie) → `AUDIT` (artefato só
relata) → `ENFORCED` (artefato bloqueia ou produz por construção) →
`DEPRECATED` / `SUPERSEDED por RN_…`.

**Critério de `ENFORCED` neste catálogo:** a regra é imposta **no caminho
de produção**. Função de especificação com teste e **sem chamador fora de
teste** não conta. Dublê ou `Noop` injetado em produção (lista vazia,
limite zero, *checker* que aceita tudo) também não — o código parece
impor, mas nada recusa. Em 2026-10-22, o inventário em `index.md` separa
os três estados pela medição direta.

## 3. Template (uma regra por seção, no arquivo da área)

```markdown
### RN_AREA_NNN — Título curto em forma de obrigação

| Campo | Valor |
|---|---|
| **Code** | `RRUL.DDD.FFF.01` · alias `RN_AREA_NNN` |
| **Status** | PROPOSED / AUDIT / ENFORCED |
| **Origem** | milestone, commit ou shape SHACL em que a regra nasceu |
| **Regra** | Uma frase imperativa e verificável |
| **Racional** | Por quê (risco, norma — Lei 14.286, Res BCB, SHACL *shape*) |
| **Imposta por** | caminho do artefato no caminho de produção |
| **Evidência** | teste ou comando que prova |
| **Exceções** | dono e prazo, ou "nenhuma" |
| **Fluxos** | RFLW.… que a aplicam |
| **Desde** | data |
```

## 4. Versionamento e índice

`VERSION` + `CHANGELOG.md` (semver: área nova = MAJOR; regra nova = MINOR;
correção = PATCH). O PR não faz bump nem escreve no `CHANGELOG.md`: deixa
um fragmento em `changelog.d/` (`AAAA-MM-DD-<slug>.md`), e a versão só
muda num release do catálogo. O `index.md` lista todas as regras com
código, alias e status; o próximo alias por área **e** o próximo `FFF`
por subdomínio são alocados lá.

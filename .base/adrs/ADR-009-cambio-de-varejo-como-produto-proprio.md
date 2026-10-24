# ADR-009: ExchangeOS passa a ter dois produtos coexistentes — interbancário (atual) e câmbio de varejo (novo, originado pelo app)

**Code:** `RADR.024.009.01` · alias `ADR-009`
**Status:** DRAFT — conteúdo decidido pelo dono em 2026-10-23 (contraparte, enquadramento, pares, limites, eventos, rotas); aguarda revisão final do documento antes de ir a `PROPOSED`

| | |
|---|---|
| **Data** | 2026-10-23 |
| **Contexto** | Decisão C3 do dono (2026-10-22) em `onboardos-revenu-integracao-mobile/.base/plans/integracao-mobile-tempo-real/README.md`: câmbio entra no app desde o início · pré-requisito de `MS-163 2.3` (`RTSK.080.823.01`) do programa `integracao-mobile-tempo-real` |
| **Supersede** | Nada |
| **Relacionados** | `ADR-002` (`fxtr`-only na borda) · `ADR-007` (dual-ledger: `ledgeros` como autoridade contábil) · § 3.3.3 do baremetal (treasuryos: FX de mesa é 283) · MS-163 2.2 (outbox → Kafka de `quote.*`/`trade.*`) · MS-164 7.2 (grupo `/mobile/v1/exchangeos` no gateway) · MS-165 fase 9 (posse no módulo + teste IDOR) · **MS-200..202** `limites-de-operacao` (RiskOS como gestor único de limites — a `MS-201 3.5` já cita esta ADR) |

## Contexto

O exchangeos foi desenhado interbancário: o agregado `FXTrade` em
`modules/trade/domain/fxtrade.go` carrega `buyer_bic` e `seller_bic`, a
borda emite `fxtr.*` CLS/CFETS com BAH, e o `settlement_venue` escolhe
PvP CLS. Não há cliente final no modelo.

Em paralelo, três fatos documentados em `MS-163-os-produtores-de-produto.md`
e no levantamento `03 § 2` do programa `integracao-mobile-tempo-real`
(medidos em 2026-10-22):

- `internal/eventbus/eventbus.go:1-11` é um «stop-gap until the Kafka outbox lands» — publicação `quote`/`trade` acontece em barramento em processo, síncrono;
- `deploy/kafka/topics.yaml:100-115` declara o `wss-gateway` consumindo `/ws/exchange/*` — tópicos que **não existem**;
- `xfxt` (business area da plataforma) tem **0 mensagens** registradas;
- o módulo não tem grupo no gateway (`06 § 2.7`): nenhuma rota `/mobile/v1/exchangeos`.

A **decisão C3** do dono (2026-10-22) resolveu que câmbio entra no app
desde o M1 do programa mobile. Como o modelo atual não tem cliente final,
MS-163 2.2 (publicar `quote.*`/`trade.*` via outbox com `customer_id`,
`event_id` e `version` no envelope) não pode acontecer: o `FXTrade`
interbancário não tem cliente para carimbar. Esta ADR fecha o contrato
de **produto** antes da MS-163 2.2 começar.

### Estado medido no app iOS (2026-10-23)

Hoje o cliente só vê câmbio pelo app, e **os limites são locais, no
dispositivo**:

- `app-revenu-ios/SDK/Core/Sources/Exchange/ExchangeCalculator.swift:5`
  declara `limitTotalCents = 100_000_000` — teto de **R$ 1.000.000 por
  operação**, hard-coded, sem par com nenhum servidor; a validação
  acontece em `ExchangeCalculator.validate(...)` linha 60.
- `iofRate = 0.011` e `serviceFeeRate = 0.005` (linhas 6-7) também vivem
  no app, como se o cliente conhecesse a alíquota do IOF antes do servidor
  computar.
- `ExchangeStore.swift` liquida o deal **no aparelho**, sem persistência
  no exchangeos.

Essa é a dívida que esta ADR resolve do lado servidor: **o teto, as
cotas, o IOF e a tarifa passam a vir do servidor**; o app deixa de
decidir e passa a mostrar o que o servidor retornou.

## Decisão

ExchangeOS passa a ter **dois produtos** no mesmo módulo:

1. **Interbancário** (o atual) — `TradeKind.INTERBANK`. Mantém BIC-a-BIC,
   `fxtr.*` CLS/CFETS, aggregates em `modules/trade/`, `modules/cfets_*`,
   `modules/cls_settlement/`. Não muda com esta ADR.
2. **Câmbio de varejo** (novo) — `TradeKind.RETAIL`. Cliente final
   originado pelo app Revenu. Contraparte, pares, limites, eventos e
   rotas abaixo.

### 1. Contraparte do cliente

**A Revenu, como principal, via mesa `treasuryos` para hedge.**

- O cliente do varejo negocia **contra a Revenu**. O exchangeos escritura
  o deal retail e emite os eventos ao app.
- O hedge interbancário é fechado pelo `treasuryos` (DDD 283, FX de mesa).
  O exchangeos **cita, não reescreve** a decisão de mesa: envia um pedido
  canônico de hedge via gRPC `treasuryos.v1.MeasureFXDesk/HedgeRetailLeg`
  e persiste a referência no agregado retail.
- A fronteira já está escrita em `baremetal/.base/plans/CONVENTIONS.md`
  § 3.3.3: **FX de mesa é do treasuryos**; câmbio de cliente é do
  exchangeos.
- Capital próprio e autorização BCB IP de pagamento (não corretor de
  câmbio): a Revenu é IP, não casa de câmbio independente.

### 2. Enquadramento regulatório

**eFX, Res. BCB 561 (vigente 01/10/2026)** — USD 10k por operação, 100%
digital, assinatura eletrônica, *disclaimer* de taxa.

- Regra `RN_FX_029` (eFX limit USD 10k investimento) já declarada em
  `.base/rules/comp.md` (quando chegar o alias) é o teto por operação.
- **Fora de escopo nesta ADR**: Res. BCB 277 (câmbio tradicional até
  USD 500k), SCE-*, DEC acima de USD 10k. Esses viram ADR separada
  quando o produto crescer.

### 3. Pares (M1)

**`USD/BRL`.** Único par no M1.

- Lista em `modules/refdata/domain/currency_pair.go`, marcado
  `kind = RETAIL`.
- `EUR/BRL`, `GBP/BRL` e demais ficam como **questão aberta**; cada um
  entra por ADR de subdomínio em 221 (trade) quando a mesa validar
  capacidade e os limites por par.

### 4. Limites

Dentro do teto eFX, três limites compostos:

| Limite | Valor M1 | Fundamento |
|---|---|---|
| Por operação | **USD 10k** | Res. BCB 561, `RN_FX_029` — teto regulatório |
| Por cliente/dia | **USD 10k** | Produto (M1 começa conservador; cliente faz uma por dia) |
| Por cliente/mês | **USD 50k** | Produto (benchmark de concorrentes mobile; configurável por refdata) |

- Limites por cliente vivem em refdata, não hard-coded: `refdata.RetailLimits{currency_pair, per_op, per_day, per_month}`.
- `RN_FX_015` (NOP realtime) continua valendo e **se soma** aos limites por cliente — se a mesa estiver no teto NOP, novo deal retail é recusado pela regra agregada.
- **Fracionamento proibido** (Res. BCB 277 arts. 29, 49): o cliente **não** pode dividir uma intenção acima do teto em duas operações menores no mesmo dia para burlar o eFX. O cálculo da cota diária é por janela de 24 h, não por «uma operação aceita, zero contador». Teste de aceite: `TestRetailFX_SplitOperationRefused` (origem: MS-201 3.5).

**Forward reference (não decidido nesta ADR).** O plano
`onboardos-revenu-limites/.base/plans/limites-de-operacao` (MS-200..202,
aguardando OK do dono) propõe que os três limites deste parágrafo virem
dimensões do **gestor único de limites no RiskOS**, consultado pelo
exchangeos antes de confirmar o trade (falha fechada). A `MS-201 3.5`
cita esta ADR explicitamente. Enquanto esse plano não for aprovado, os
limites continuam em `refdata.RetailLimits`; quando for, a ADR seguinte
em 227 (risco) ou 024 (governança) decide a migração e o `TradeKind.RETAIL`
passa a consultar `riskos.v1.LimitsManager/Check`. O `RN_FX_015` (NOP
tesouraria) **permanece no exchangeos** mesmo após a migração.

### 5. Rotas HTTP (grupo `/mobile/v1/exchangeos` da MS-164 7.2)

Quatro rotas. **Identidade sempre do token** (`sub` / `person_entity_id`),
**nunca do corpo** — pré-requisito do teste IDOR da MS-165 fase 9.

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/pairs` | Lista pares disponíveis ao cliente (hoje USD/BRL) + os três limites efetivos (per_op, per_day, per_month) descontando o que já consumiu |
| `POST` | `/quote` | Pede cotação (RFQ). Resposta: `{quote_id, rate, expires_at, amount_quote, amount_base}`. Validade curta (p.ex. 15s) |
| `POST` | `/trades` | Aceita `{quote_id}`. Resposta: `{trade_id, status: PENDING}`. Validações: cotação não expirou, limites não excedidos, sanções triadas (`RN_FX_039`), contraparte real (mesa não congelada) |
| `GET` | `/trades/{id}` | Status atual do trade (`PENDING`, `CONFIRMED`, `SETTLED`, `REJECTED`) e, quando SETTLED, referência de liquidação no ledgeros |

**Corpo não carrega `customer_id` nem `tenant_id`.** O gateway valida o
token, injeta como *claims* resolvidas, e o handler lê `ctx` — nunca o
JSON. Isso é condição de posse para `MS-165 fase 9`.

### 6. Eventos ao vivo (envelope MS-163 2.2)

Cinco eventos canônicos, publicados por **outbox transacional** → Kafka,
no envelope da plataforma (`aggregate_id`, `version`, `customer_id` /
`entity_id`, `event_id`, `tenant_id`):

| Evento | Dispara | Payload mínimo |
|---|---|---|
| `quote.offered` | `POST /quote` retorna sucesso | `quote_id`, `rate`, `expires_at`, `amount_quote`, `amount_base`, `pair` |
| `quote.expired` | `RFQ.Expire()` disparado por timer quando `expires_at` passa sem aceite | `quote_id` |
| `trade.confirmed` | `POST /trades` aceita + mesa confirmou o hedge no treasuryos | `trade_id`, `quote_id`, `rate`, `amount_quote`, `amount_base`, `pair` |
| `trade.settled` | `ledgeros` confirma o lançamento | `trade_id`, `ledger_entry_id` |
| `trade.rejected` | `POST /trades` recusado (quote expirada, limite excedido, sanção, mesa congelada) | `trade_id`, `reason` |

- **Tópicos Kafka**: `revenu.xfxt.quote.*` e `revenu.xfxt.trade.*`
  (derivados do identificador no catálogo — não constantes de código).
- `xfxt` precisa ter mensagens registradas no catálogo (MS-163 2.2 §
  aceite: `ledgeros/messages/xfxt.*/definition.yaml` com `eventos:` não
  vazio).
- O `wss-gateway` consome **pelo canal do cliente** (MS-158), não por
  `/ws/exchange/*` (que `topics.yaml:100-115` declarava e não existia —
  essa linha é dívida que esta ADR fecha).

### 7. Fronteira com o interbancário e com o treasuryos

- `FXTrade` **não é reusado tal como está**: ou ganha `TradeKind` como
  campo e um bloco `retail: {customer_id, person_entity_id}` opcional, ou
  nasce um agregado novo `RetailFXTrade` em `modules/retail_trade/`. A
  escolha entre as duas é **decisão de ADR subsequente de 221 (trade)**,
  não desta ADR.
- `treasuryos` é consumido via gRPC, não compartilha DB. Esta ADR fecha
  **o contrato de cliente**; o contrato com o treasuryos é escrito como
  ADR no DDD 283 do treasuryos ou DDD 221 daqui, não aqui.

## Consequências

**Positivas:**

- MS-163 2.2 destrava: há cliente, agregado e eventos para publicar.
- MS-164 7.2 destrava: há rotas definidas e condição de IDOR explícita.
- MS-165 fase 9 ganha um critério objetivo de IDOR: nenhum endpoint
  aceita `customer_id` no corpo.
- A fronteira exchangeos ↔ treasuryos deixa de ser implícita.

**Negativas / trade-offs:**

- Dois produtos no mesmo módulo aumentam a área cognitiva. Mitigação:
  `TradeKind` explícito e teste que recusa `RETAIL` sem `customer_id` e
  `INTERBANK` sem `buyer_bic`/`seller_bic`.
- Hedge no treasuryos cria acoplamento gRPC: se o treasuryos cair, o
  `POST /trades` recusa. É o comportamento certo (não aceitar deal sem
  hedge), mas precisa de SLO e circuit breaker — detalhado em ADR de 227
  (risco).
- `xfxt` com 0 mensagens no catálogo é um defeito preexistente que a
  MS-163 2.2 fecha; esta ADR só cita.

## Alternativas consideradas

| Alternativa | Por que foi rejeitada |
|---|---|
| Banco parceiro como contraparte (treasuryos só roteia) | Depende de contrato bilateral que ainda não existe; adiciona 3-6 meses de negociação; mantém a Revenu como corretor sem balanço, o que compromete margem. Pode voltar numa ADR futura. |
| Market-maker externo (EBS, 360T) | Exigiria autorização de corretor de câmbio no BCB, além do IP de pagamento que a Revenu já tem. Fora do escopo regulatório atual. |
| Enquadrar em Res. BCB 277 (até USD 500k) | Exige código de natureza (`RN_FX_028`, 95 códigos), DEC acima de USD 10k, presença de gerente para montantes grandes — fora do que o app resolve 100% digital. |
| Suite de 3 pares (USD/BRL + EUR/BRL + GBP/BRL) no M1 | Risco de mesa de hedge em EUR e GBP não calibrada; cada par tem comportamento próprio (liquidez, horário, PTAX). M1 conservador com USD/BRL. |
| Reusar `FXTrade` interbancário sem distinção | O agregado exige `buyer_bic` e `seller_bic`; o cliente de varejo não tem BIC. Forçar BIC de fachada polui o modelo e invalida o `RN_FX_010` (CLS para pares elegíveis — retail não vai para CLS). |
| Deixar nomes de evento como `OPEN` até a MS-163 2.2 | MS-163 2.2 precisaria entrar na ADR; ping-pong. Melhor fixar aqui o vocabulário e deixar a 2.2 implementar. |

## Verificação

Como saber que a decisão está implementada (preencher ao mover para `ACCEPTED`):

- `grep -n "TradeKind" modules/` mostra o enum com `INTERBANK` e `RETAIL`;
- `curl` nas 4 rotas de `/mobile/v1/exchangeos` com token válido retorna 200/202; com `customer_id` no corpo e token de outro cliente, retorna 403 (teste IDOR da MS-165 fase 9);
- `kafka-console-consumer --topic revenu.xfxt.quote.offered` recebe eventos com `aggregate_id`, `version`, `customer_id`, `event_id`, `tenant_id` populados;
- `ledgeros/messages/xfxt.quote.*/definition.yaml` e `.../trade.*/definition.yaml` com `eventos:` não vazio;
- `TestRetailTradeRejectedBeyondDailyLimit` (novo), `TestRetailFX_SplitOperationRefused` (novo, origem MS-201 3.5) e `TestInterbankTradeRejectsRetailFields` (novo) passam;
- `deploy/kafka/topics.yaml` não declara mais `/ws/exchange/*` morto;
- `app-revenu-ios/SDK/Core/Sources/Exchange/ExchangeCalculator.swift` perde `limitTotalCents`, `iofRate` e `serviceFeeRate` como constantes locais — passa a ler esses valores do servidor via `GET /mobile/v1/exchangeos/pairs` e `POST /mobile/v1/exchangeos/quote`.

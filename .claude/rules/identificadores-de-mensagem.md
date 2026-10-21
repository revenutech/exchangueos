# Regra: Identificadores de mensagem não se substituem

> **Nota de frescor 2026-09-17 (RI-1371).** O incidente MS-057 citado adiante ('o wss-gateway mapeia 24 eventos canônicos... e o catálogo não o declara como consumidor') está ENCERRADO: o catálogo hoje declara o `wss-gateway` como `consumidores:` em ~139 definições (nunca como emissor), e o `event_mapper.go` mapeia ~120 entradas (não 24). A safra RI-1355→1365 ligou 125 eventos ao catálogo do gateway via MustID. O trecho adiante é narrativa histórica.


> **Quick Reference:** um identificador `AAAA.NNN.NNN.NN` significa **uma**
> mensagem. Nem o da ISO 20022 nem o próprio da Revenu podem receber semântica
> diferente da registrada. Vale para os dois igualmente.
> O envelope segue a mesma lógica: header divergente entrega o mesmo
> resultado que identificador errado — ninguém recebe.
> **Fonte:** ADR-060 e ADR-061 em `ledgeros/.base/plans/00-governance/decision-records/` · catálogo em `ledgeros/.base/aasc/iso20022/`

## Princípio

O identificador **é** o contrato. `acmt.019.001.04` é AccountClosingRequest para
qualquer sistema do mundo que fale ISO 20022, e `blkg.008.001.01` é o início de
bloqueio judicial para qualquer sistema da Revenu.

Reaproveitar um identificador para outra coisa não é escolha de nome: é fazer o
contrato mentir.

## A premissa, e o que a torna verificável

A regra acima é sobre o identificador. A premissa que ela serve é maior, e está
no `README.md` do catálogo:

> **Um fato de negócio tem UMA definição, independentemente de quantos módulos o
> emitem.** (`RN_MSG_004`)

O emissor não entra no identificador — vive no BAH, campo `Fr`. É isso que
permite uma definição servir a vários módulos em vez de gerar uma por emissor.

**Três camadas sustentam a premissa, e elas não têm a mesma força:**

| camada                               | quem garante                                  | força                                                                  |
| ------------------------------------ | --------------------------------------------- | ---------------------------------------------------------------------- |
| unicidade do **identificador**       | um diretório por identificador                | inviolável                                                             |
| duas definições com o mesmo **nome** | `RN_MSG_004`, igualdade de string normalizada | heurística — não pega duplicação semântica, e isso é julgamento do SEG |
| o nome do **evento**                 | campo `eventos:` + `msgrepo events`           | desde 2026-08-01                                                       |

A terceira não existia, e é a que liga o catálogo ao código. `definition.yaml`
declarava `emissores` e `consumidores` — nomes de MÓDULO — e nada sobre o nome do
evento; a correspondência vivia só no mapa Go de cada módulo, e todo comando do
`msgrepo` recebia `--module`. **Nenhum cruzava dois módulos, então nenhum podia
comparar.**

Ao registrar mensagem, declare `--events` com o nome canônico do fato no código.
Sem ele o catálogo sabe que a mensagem existe e não sabe que fato ela representa.

## As duas metades da regra

### 1. Domínio ISO 20022 nativo — `AAAA.NNN.NNN.NN`

| Proibido                                                                                | Por quê                                                                     |
| --------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Declarar `urn:iso:std:iso:20022:tech:xsd:X` para XML que não valida contra o XSD de `X` | Uma contraparte que resolva o namespace espera uma mensagem e recebe outra  |
| Usar um identificador ISO com significado próprio                                       | O identificador é global; o significado não é nosso para redefinir          |
| Alocar identificador dentro de business area da ISO                                     | `RN_MSG_008`. A ISO pode alocar o mesmo número amanhã                       |
| Trocar a versão sem trocar o identificador                                              | `AAAA.NNN.NNN.NN` — o último par É a versão. Mudança incompatível vira `+1` |

**Obrigatório:** mensagem ISO entra como **classe A**, adotada por referência, com
o XSD oficial versionado e verificável. `msgrepo xsd` confere.

### 2. Domínio próprio da Revenu

**A mesma regra, sem exceção.** `blkg.008.001.01` não é menos contrato por ser
nosso — é contrato entre nossos módulos, e trocá-lo por baixo quebra consumidor
do mesmo jeito.

| Proibido                                               | Por quê                                                                                              |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------- |
| Reaproveitar identificador `WITHDRAWN`                 | Um consumidor antigo interpretaria a mensagem nova com o schema velho                                |
| Mudar o significado de um identificador registrado     | Registrar é publicar. Republicar outra coisa no mesmo endereço é o defeito                           |
| Criar uma segunda definição para um fato já registrado | `RN_MSG_004` — um fato, uma definição. O emissor vive no BAH, não no identificador                   |
| Usar o nome do evento como nome de tópico e vice-versa | São coisas diferentes: o tópico deriva da BASE do identificador, o subject do identificador completo |

**Renomear a ÁREA é permitido** enquanto a mensagem não estiver registrada —
depois, é `msgrepo realloc`, e só até o marco de imutabilidade.

### 3. O envelope também é contrato

O identificador diz QUAL mensagem é. O envelope diz como o consumidor a
encontra. Acertar o primeiro e errar o segundo entrega o mesmo resultado que
errar os dois: ninguém recebe.

| Proibido                                                                | Por quê                                                                                                     |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Produtor e consumidor com nomes de header diferentes para o mesmo campo | O consumidor lê, não acha, descarta. Ver ADR-061                                                            |
| Descartar mensagem por header ausente em log `Debug`                    | Em `info` — o padrão — o descarte é invisível. Offset comita, lag zera, e o dado some sem sinal             |
| Publicar sem `msg_def_idr`                                              | É o identificador viajando na mensagem. Sem ele o consumidor não tem como resolver o contrato pelo catálogo |

**O caso que originou:** o `onboardos` publicava em `revenu.xkyc.001` com
headers CloudEvents (`ce_type`, `ce_source`, `ce_id`) mais `msg_def_idr`; o
`wss-gateway` lia `event_type`, `aggregate_id` e `tenant_id`. Tópico certo,
identificador certo, catálogo declarando o gateway como consumidor — e o
evento descartado em silêncio.

Nenhum sinal usual acusava: produtor logava sucesso, tópico tinha a mensagem,
o grupo de consumo mostrava `LAG 0`. O `LAG 0` era literalmente verdade — o
gateway consumia e jogava fora.

## O que fazer quando não existe identificador certo

Não force. Nesta ordem:

1. **Existe mensagem ISO que cobre o fato?** → classe **A**, por referência.
2. **Existe mensagem ISO parecida, mas o conteúdo é nosso?** → classe **B** em
   área própria, com `relatedTo` apontando a ISO. O parente fica declarado, e o
   namespace é `urn:revenu:msg:xsd:`.
3. **Não existe nada parecido?** → classe **C**, área própria, sem `relatedTo`.

O erro que esta regra existe para impedir é pular direto para "uso o
identificador da ISO que mais parece".

## Dois incidentes reais que originaram a regra

**InvestOS (MS-058 §20).** Catorze de dezessete declarações emitiam elemento raiz
inexistente no schema declarado. `secl.010` dizia `ClrMmbrRpt` quando o oficial é
`SttlmOblgtnRpt`; `reda.014` dizia `SctiesRpt` quando aquele identificador é
`PartyCreationRequest` — mensagem de outro assunto. O XML declarava o namespace
da ISO e não validava contra ele.

O defeito era **latente**: nada importava o pacote, então nenhum XML saía. Viraria
incidente no dia em que alguém ligasse o builder a um adaptador.

**LedgerOS × AccountOS (MS-057 §13).** `account.opening_requested` é canônico no
`accountos`, onde carrega `acmt.007.001.05`, e canônico no `ledgeros`, onde não
carrega identificador nenhum e sai em `ledgeros.account`. A absorção do MS-048
trouxe as mesmas 52 constantes de evento com zero referências a `iso20022`.

**Os dois gates ficaram verdes**, porque cada um olhava um módulo: o `RN_MSG_016`
confere que o identificador aparece no código do `accountos`, e aparece. Um
consumidor que assine `revenu.acmt.007` não recebe nada do binário do LedgerOS.

Junto com ele, o espelho: o `wss-gateway` mapeia 24 eventos canônicos em
`event_mapper.go` e o catálogo não o declara como consumidor. Havia gate para
consumidor declarado que não consome (`RN_MSG_019`) e nenhum para o contrário.

**AuthorityOS (MS-058 §19).** Transferência de reservas modelada em três lugares,
duas delas com entidades Go distintas dentro do mesmo módulo, e sem UETR. A
pergunta "esta transferência já aconteceu?" tinha duas respostas possíveis.

## Como verificar

```bash
# O identificador existe e o XSD confere (classe A)
cd ledgeros/tools/msgrepo && go run . xsd --repo ../../.base/aasc/iso20022

# As invariantes do catálogo, incluindo RN_MSG_004 e RN_MSG_008
go run . validate --repo ../../.base/aasc/iso20022

# O código não emite evento fora do catálogo
go run . drift --module <módulo> --src <caminho> --repo ../../.base/aasc/iso20022

# O MESMO evento em dois módulos — a única varredura cruzada do CLI.
# Todo comando acima recebe `--module` e olha um só; este lê o modules.yaml
# e cruza os 16 repositórios.
go run . events --repo ../../.base/aasc/iso20022
```

Nenhum desses confere ENVELOPE — todos param no identificador. A divergência
de header do ADR-061 passou por todos eles em verde. Enquanto não houver gate,
a verificação é observar o consumidor de fato receber:

```bash
# Os tópicos que o gateway assina, incluindo os derivados do catálogo, que
# são computados e não aparecem como literal no fonte.
cd wss-gateway-revenu/src && go run ./cmd/topics

# O grupo consumiu? LAG 0 com o cliente sem receber é o sintoma do ADR-061.
# 🔴 AS DUAS RÉGUAS ABAIXO ESTAVAM CADUCAS — medido 2026-09-30, e as duas
# falhavam de maneira que se LÊ COMO RESULTADO:
#
#   `:9094` → o broker desta bancada atende texto simples em **:9092**; a 9094 é
#   SASL desde o piloto de SCRAM, e ali o comando morre num **timeout** que
#   se lê como «broker em baixo». Medido: :9092 lista tópicos, :9094 expira.
#
#   `wssgateway-<módulo>` → os grupos são por **DOMÍNIO**, não por módulo:
#   `wssgateway-onboarding`, `-identity`, `-ledger`, `-charge`… (14 medidos).
#   Procurar `wssgateway-onboardos` devolve **nada**, e o nada lê-se como
#   «não há consumidor».
#
# ⚠️ E as ferramentas do broker precisam de `KAFKA_OPTS=` neutralizado: com o
# valor herdado, a listagem morre num rastreio de Java e imprime **0 tópicos**
# — e o zero é a MEDIÇÃO a falhar, não o broker a estar vazio.
#
# 🔴 E a régua que este bloco existe para dar continua a NÃO servir sozinha:
# `LAG 0` mede CONSUMO, nunca ENTREGA. Medido ao vivo a 2026-09-30: plantadas
# 6 mensagens, o grupo consumiu **as 6 com LAG 0 e offset a andar**, e **2**
# chegaram ao cliente WebSocket. As outras quatro foram descartadas por
# envelope divergente, com o corpo na dead-letter a CONTER o nome do evento
# que o consumidor procurava no header. ***A informação estava dentro da
# mensagem e inalcançável.***
#
# A única prova de entrega é um cliente a RECEBER o frame.
docker exec <kafka> /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 --describe --group wssgateway-<DOMINIO>
# (e a listagem dos grupos reais:)
# docker exec <kafka> sh -c 'KAFKA_OPTS= /opt/kafka/bin/kafka-consumer-groups.sh \
#   --bootstrap-server localhost:9092 --list' | grep wss
```

`events` reporta quatro coisas: **E1** o mesmo evento em dois identificadores ·
**E2** evento no código de módulo que não é emissor nem consumidor · **E3**
emissor declarado cujo código não contém a string · **E4** definição com emissor
e `eventos: []`.

O `MustID` do `revenu-common` derruba o processo no init se o identificador não
estiver no catálogo. É deliberado: processo que sobe emitindo mensagem fora do
catálogo é pior que processo que não sobe.

## Anti-patterns

| Evitar                                              | Preferir                                                                                          |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `urn:iso:...:reda.014` para um relatório de títulos | `reda.012`, que **é** SecurityReport                                                              |
| Escolher o identificador pelo nome que soa parecido | Conferir o elemento raiz contra o XSD oficial                                                     |
| Inventar área de 4 letras dentro do espaço da ISO   | Área própria declarada em `business-areas.yaml`                                                   |
| Constante de tópico com o texto do tipo de evento   | Tópico derivado do identificador                                                                  |
| Reusar número de mensagem retirada                  | Alocar o próximo; retirado não volta                                                              |
| Registrar sem `--events`                            | Declarar o fato que a mensagem representa — sem isso o catálogo não tem como cruzar com o código  |
| Copiar constantes de evento de outro módulo         | Consumir a definição; duas cópias do mesmo fato divergem sem nada acusar                          |
| Consumidor lendo header com nome próprio            | Ler o que o catálogo manda carimbar — `msg_def_idr` resolve o contrato; nome de evento é derivado |
| Tratar mensagem descartada como caso normal         | Consumir e jogar fora é anomalia. `Warn`, não `Debug` — senão some sem sinal                      |

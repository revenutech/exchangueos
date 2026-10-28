# PTN-NNN: <nome do padrão>

**Code:** `RPTN.091.NNN.01` · alias `PTN-NNN`
**Estado:** PROPOSTO | PROPOSTO, sem exemplo | ACEITE | SUBSTITUÍDO | RETIRADO — <data em BRT e o porquê do estado>
**Tema:** eda | ddd | golang | cockroachdb | kafka | grpc | testes | seguranca | …
**Âmbito:** plataforma (transversal) | <módulo>
**Substitui / redireciona:** <códigos ou aliases antigos, por exemplo `cardos PTN-002`, `CD-008`; ou «—»>
**Correção:** <os FIX que restauram este padrão, por exemplo `FIX-006 (RFIX.053.006.01, ledgeros) — Concorrência otimista`; ou «—». O RFIX fica fora da precedência (O162)>

<!--
Template do ADR-018 (RADR.091.018.01). As dez secções abaixo são obrigatórias e ficam por esta ordem.
Um padrão descreve, não obriga (RN_CTX_027): sem título «Regra:» e sem «deve»/«nunca»/«sempre»/«must».
O que tem de ser seguido escreve-se como regra RRUL que obriga este padrão, e lista-se em «Regras que obrigam».
-->

## Problema

<A situação concreta em que o problema aparece, em uma ou duas frases.>

## Contexto e forças

<As forças em tensão, por exemplo consistência × disponibilidade, ou acoplamento × visibilidade. Inclui os limites medidos, se houver.>

## Solução

<A forma da solução, com a estrutura mínima: tipos, tabelas e sequência. O código daqui ilustra; o exemplo real é a secção própria.>

## Consequências

<O que se ganha, o que se paga e o que passa a ser preciso operar.>

## Quando não usar

<Os casos em que este padrão é a escolha errada, e o que se usa nesses casos.>

<!-- Secção opcional «## Divergência», aqui, só quando o código desmente a origem: o padrão descreve o que o código faz e a divergência cita a medição e a base de conhecimento (O112). -->

## Exemplo real

<!-- RN_CTX_026: medido no código, nunca de memória. Um exemplo sem chamador de produção marca-se «sem chamador». -->

| Repo | Caminho | Símbolo | sha | Chamadores de produção |
|---|---|---|---|---|
| exchangeos | <caminho>:<linha> | <tipo ou função> | <sha curto> | <n e um caminho> |

## Fundamentação

| Fonte | Caminho em `knowledge/` | Página ou âncora |
|---|---|---|
| <título> | `knowledge/plataforma/<tema>/…` | <!-- p. N --> |

## ADR que adota

- `RADR.DDD.FFF.VV` (<alias>, exchangeos) — <estado do ADR>

## Regras que obrigam

- `RRUL.DDD.FFF.VV` (<alias>) — <a cláusula que obriga este padrão>; ou «nenhuma: o padrão é recomendado, não obrigatório»

## Antipadrões

| Antipadrão | Consequência | Alternativa |
|---|---|---|
| <o que se vê> | <o que causa> | <o que este padrão faz em vez disso> |

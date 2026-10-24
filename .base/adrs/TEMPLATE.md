# ADR-NNN: Título curto, em forma de decisão

<!--
Convenções:

1. O campo `Status` tem UMA forma: `**Status:** X` no início de linha, valor em
   MAIÚSCULAS do vocabulário fechado. Não use seção `## Status`, tabela,
   citação `> Status:` nem item de lista.

2. Qualificação depois do valor é bem-vinda: `**Status:** PROPOSED — aguardando
   medição de banda em homologação`.

3. O número (`ADR-NNN`) e o código `RADR.DDD.FFF.VV` saem do `index.md` no
   momento da alocação. NÃO fixe um número «reservado» antes disso.

4. Aceito é imutável. Para mudar, escreva uma ADR nova que SUPERSEDE esta e
   atualize o status das duas.

Apague este comentário ao criar a ADR.
-->

**Code:** `RADR.DDD.FFF.01` · alias `ADR-NNN`   <!-- DDD do registro único (baremetal/.base/plans/CONVENTIONS.md § 3.3.5); FFF alocado no index.md -->
**Status:** PROPOSED

<!-- Vocabulário: DRAFT | PROPOSED | ACCEPTED | DEPRECATED | SUPERSEDED | REJECTED -->

| | |
|---|---|
| **Data** | YYYY-MM-DD |
| **Contexto** | Fase/plano/commit que motivou a decisão |
| **Supersede** | Nada · ou `ADR-NNN` |
| **Relacionados** | ADRs, fluxos, regras, planos |

## Contexto

Qual problema ou força está motivando esta decisão? O que se mediu ou observou?
Cite números, limites, requisitos — não intenções.

## Decisão

O que vamos fazer, em frases afirmativas. Se houver configuração concreta
(versões, parâmetros, topologia, caminhos), coloque aqui — é o que alguém
vai copiar.

## Consequências

**Positivas:**

- …

**Negativas / trade-offs:**

- …

## Alternativas consideradas

| Alternativa | Por que foi rejeitada |
|---|---|
| … | … |

## Verificação

Como saber que a decisão está de fato implementada (comando, métrica, estado
esperado). Preencher ao mover para ACCEPTED.

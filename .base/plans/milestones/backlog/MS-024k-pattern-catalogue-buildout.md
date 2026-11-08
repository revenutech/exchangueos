# MS-024k — Pattern Catalogue Build-out (850 patterns)

**Code:** `RMLS.024.017.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=17; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Distributed — one pattern per code review
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024k/plano-original-shape-1.1.0.md](../registro/ms-024k/plano-original-shape-1.1.0.md)

## Por quê

`.base/plans/01-architecture/patterns/` indexa 20 catálogos totalizando 850 padrões planejados. Hoje 27 padrões têm Context/Problem/Solution/Example/Anti-pattern/Related completos; os 823 restantes são placeholders nomeados. Catálogo com > 95% placeholder é documentação-teatro.

## Descrição

Converter os 823 placeholders em entradas reais ao longo do ciclo MS-024, usando imposto "pattern-per-PR": qualquer PR tocando área de padrão documentado deve preencher o padrão ou abrir dívida rastreada. Meta: 300 padrões totalmente escritos até fim do ciclo (de 850), priorizando os mais citados (FX-GP-*, FX-DDD-*, FX-EDA-*, FX-CP-*, FX-COMMIT-*).

## Fase 1 — Implementação

### Etapa 1.1 — Template + lint

- **1.1** `RTSK.024.1701.01` ✅ Formalizar template em `patterns/TEMPLATE.md` (6 seções, ≤ 400 palavras cada). *Aceite:* template usado como base para 1 pattern de referência. *Entrega:* `.base/plans/01-architecture/patterns/TEMPLATE.md`. *Escopo:* `.base/plans/01-architecture/patterns/`. [registro](../registro/ms-024k/README.md)
- **1.2** `RTSK.024.1702.01` ✏️ Lint script `scripts/lint-patterns.sh` validando 6 seções + ≥ 1 exemplo de código + ≥ 1 anti-pattern + ≥ 1 cross-link por arquivo. *Aceite:* script falha em pattern incompleto, passa em pattern completo. *Entrega:* `scripts/lint-patterns.sh`. *Escopo:* `scripts/`. *Depende:* 1.1. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.3** `RTSK.024.1703.01` ✏️ Lefthook pre-commit rodando pattern lint quando `patterns/` muda. *Aceite:* hook reprova commit com pattern sem 6 seções. *Entrega:* atualização em `lefthook.yml`. *Escopo:* `lefthook.yml`. *Depende:* 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Preenchimento por família

- **1.4** `RTSK.024.1704.01` ✏️ 60 patterns FX-GP-* preenchidos. *Aceite:* 60 arquivos passam `lint-patterns.sh`. *Entrega:* arquivos em `.base/plans/01-architecture/patterns/general-purpose/`. *Escopo:* `patterns/general-purpose/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.5** `RTSK.024.1705.01` ✏️ 50 patterns FX-DDD-* preenchidos. *Aceite:* 50 arquivos passam lint. *Entrega:* `patterns/ddd/`. *Escopo:* `patterns/ddd/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.6** `RTSK.024.1706.01` ✏️ 50 patterns FX-EDA-* preenchidos. *Aceite:* 50 arquivos passam lint. *Entrega:* `patterns/eda/`. *Escopo:* `patterns/eda/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.7** `RTSK.024.1707.01` ✏️ 40 patterns FX-CP-* preenchidos. *Aceite:* 40 arquivos passam lint. *Entrega:* `patterns/compliance/`. *Escopo:* `patterns/compliance/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.8** `RTSK.024.1708.01` ✏️ 30 patterns FX-COMMIT-* preenchidos. *Aceite:* 30 arquivos passam lint. *Entrega:* `patterns/commit/`. *Escopo:* `patterns/commit/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.9** `RTSK.024.1709.01` ✏️ 30 patterns FX-QA-* preenchidos. *Aceite:* 30 arquivos passam lint. *Entrega:* `patterns/qa/`. *Escopo:* `patterns/qa/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.10** `RTSK.024.1710.01` ✏️ 40 patterns miscelâneos preenchidos. *Aceite:* 40 arquivos passam lint. *Entrega:* diversas pastas de `patterns/`. *Escopo:* `patterns/`. *Depende:* 1.1, 1.2. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Cross-link + progresso

- **1.11** `RTSK.024.1711.01` ✏️ Cada pattern preenchido referencia ≥ 1 caminho de código produção (`file:line`). *Aceite:* `grep -rE 'file:[0-9]+' patterns/` encontra ≥ 300 referências. *Entrega:* refs dentro dos arquivos de 1.4-1.10. *Escopo:* `patterns/`. *Depende:* 1.4-1.10. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.12** `RTSK.024.1712.01` ✏️ Patterns README atualizado semanal com barra de progresso. *Aceite:* README mostra `<contagem>/850` atualizado. *Entrega:* `.base/plans/01-architecture/patterns/README.md`. *Escopo:* `patterns/`. *Depende:* 1.4-1.10. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.13** `RTSK.024.1713.01` ✏️ Top-10 patterns mais citados medidos via `grep -r "FX-.*-NNN" modules/ pkg/ | sort | uniq -c | sort -rn`. *Aceite:* top-10 publicado no README. *Entrega:* seção em `patterns/README.md`. *Escopo:* `patterns/`. *Depende:* 1.12. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.4 — Governança

- **1.14** `RTSK.024.1714.01` ✏️ Pattern-review trimestral (1h) para retirar deprecated + promover úteis. *Aceite:* 1 sessão realizada com notas em `patterns/reviews/`. *Entrega:* notas de review. *Escopo:* `patterns/reviews/`. *Espera:* dono — agenda da sessão. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1715.01` ✏️ Rodar aceite da MS. *Aceite:* 300 patterns passam lint + README atualizado + 1 review trimestral realizada. *Entrega:* relatório em `registro/ms-024k/`. *Escopo:* `.base/plans/01-architecture/patterns/`. *Espera:* gate — lint patterns. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1716.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `.base/plans/01-architecture/patterns/README.md` — índice atual
- `CLAUDE.md` workflow §2 (TDD-first) consumido por patterns
- ISO 27001 controle 5.37 (documented operating procedures)

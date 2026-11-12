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
- **1.2** `RTSK.024.1702.01` ✅ Lint script `scripts/lint-patterns.sh` validando 6 seções + ≥ 1 exemplo de código + ≥ 1 anti-pattern + ≥ 1 cross-link por arquivo. *Aceite:* script falha em pattern incompleto, passa em pattern completo. *Entrega:* `scripts/lint-patterns.sh`. *Escopo:* `scripts/`. *Depende:* `RTSK.024.1701.01`. [registro](../registro/ms-024k/2026-11-10-lint-patterns-3b94.md)
- **1.3** `RTSK.024.1703.01` ⬜ `lint-patterns.sh` com base legada no pre-commit. *Aceite:* base: id + sha256 das 27 seções de 6 catálogos; só nova ou alterada é checada: nova ruim e FX-GP-002 editada reprovam; intocada passa. *Entrega:* `scripts/lint-patterns.sh`, `lefthook.yml`. *Escopo:* `scripts/`, `tests/scripts/`, `lefthook.yml`. *Depende:* `RTSK.024.1702.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.2 — Preenchimento por família

- **1.4** `RTSK.024.1704.01` ✏️ Seções FX-GP (Go) até o teto de 40 do catálogo, não 60. *Aceite:* 35 novas seções `FX-GP-006..040`, cada uma passa no `lint-patterns.sh` com a base da 1.3. *Entrega:* `patterns/200-fx-golang-patterns.md`. *Escopo:* `patterns/200-fx-golang-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`, `RTSK.024.1703.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.5** `RTSK.024.1705.01` ✏️ Seções FX-DDD até o teto de 35 do catálogo, não 50. *Aceite:* 31 novas seções `FX-DDD-005..035`, cada uma passa no `lint-patterns.sh` com a base da 1.3. *Entrega:* `patterns/201-fx-ddd-patterns.md`. *Escopo:* `patterns/201-fx-ddd-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`, `RTSK.024.1703.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.6** `RTSK.024.1706.01` ✏️ Seções FX-EDA até o teto de 45 do catálogo, não 50. *Aceite:* 41 novas seções `FX-EDA-005..045`, cada uma passa no `lint-patterns.sh` com a base da 1.3. *Entrega:* `patterns/202-fx-eda-patterns.md`. *Escopo:* `patterns/202-fx-eda-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`, `RTSK.024.1703.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.7** `RTSK.024.1707.01` ✏️ Seções FX-CP (CockroachDB; compliance é a 1.19). *Aceite:* 40 novas seções `FX-CP-NNN` (≤ 050), cada uma passa no `lint-patterns.sh` com a base da 1.3. *Entrega:* `patterns/205-fx-cockroachdb-patterns.md`. *Escopo:* `patterns/205-fx-cockroachdb-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`, `RTSK.024.1703.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.8** `RTSK.024.1708.01` ⬜ Criar o catálogo FX-COMMIT do README (teto 25), parte 1. *Aceite:* 13 seções `FX-COMMIT-001..013`; `lint-patterns.sh` sai 0 no arquivo. *Entrega:* `patterns/300-fx-precommit-enforcement-patterns.md`. *Escopo:* `patterns/300-fx-precommit-enforcement-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.17** `RTSK.024.1719.01` ✏️ Catálogo FX-COMMIT, parte 2. *Aceite:* 12 seções `FX-COMMIT-014..025`; `lint-patterns.sh` sai 0 no arquivo. *Entrega:* `patterns/300-fx-precommit-enforcement-patterns.md`. *Escopo:* `patterns/300-fx-precommit-enforcement-patterns.md`. *Depende:* `RTSK.024.1708.01`. *Espera:* sessão dona — 1.8 entregue. [registro](../registro/README.md)
- **1.9** `RTSK.024.1709.01` ⬜ Criar o catálogo FX-QA do README (teto 35), parte 1. *Aceite:* 15 seções `FX-QA-001..015`; `lint-patterns.sh` sai 0 no arquivo. *Entrega:* `patterns/260-fx-qa-tdd-e2e-patterns.md`. *Escopo:* `patterns/260-fx-qa-tdd-e2e-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.18** `RTSK.024.1720.01` ✏️ Catálogo FX-QA, parte 2. *Aceite:* 15 seções `FX-QA-016..030`; `lint-patterns.sh` sai 0 no arquivo. *Entrega:* `patterns/260-fx-qa-tdd-e2e-patterns.md`. *Escopo:* `patterns/260-fx-qa-tdd-e2e-patterns.md`. *Depende:* `RTSK.024.1709.01`. *Espera:* sessão dona — 1.9 entregue. [registro](../registro/README.md)
- **1.10** `RTSK.024.1710.01` ✏️ FX-KP e FX-DS. *Aceite:* 40 novas seções `FX-KP-NNN` (≤ 060) ou `FX-DS-NNN` (≤ 050), cada uma passa no `lint-patterns.sh` com a base da 1.3. *Entrega:* seções no Escopo. *Escopo:* `patterns/206-fx-kafka-patterns.md`, `patterns/210-fx-devsecops-cicd-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`, `RTSK.024.1703.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.19** `RTSK.024.1721.01` ✏️ Criar o catálogo FX-CMP (compliance; 204, número do plano original). *Aceite:* 15 seções `FX-CMP-001..015`; `lint-patterns.sh` sai 0 no arquivo; revisor: IdentityOS (PII). *Entrega:* `patterns/204-fx-compliance-patterns.md`. *Escopo:* `patterns/204-fx-compliance-patterns.md`. *Depende:* `RTSK.024.1701.01`, `RTSK.024.1702.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.3 — Cross-link + progresso

- **1.11** `RTSK.024.1711.01` ✏️ Cada pattern cita ≥ 1 caminho de código de produção (`file:line`). *Aceite:* ≥ 300 referências `file:line` em `patterns/`. *Entrega:* refs nos patterns. *Escopo:* `patterns/`. *Depende:* `RTSK.024.1704.01`, `RTSK.024.1705.01`, `RTSK.024.1706.01`, `RTSK.024.1707.01`, `RTSK.024.1708.01`, `RTSK.024.1709.01`, `RTSK.024.1710.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.12** `RTSK.024.1712.01` ✏️ Patterns README atualizado semanal com barra de progresso. *Aceite:* README mostra `<contagem>/850` atualizado. *Entrega:* `patterns/README.md`. *Escopo:* `patterns/`. *Depende:* `RTSK.024.1704.01`, `RTSK.024.1705.01`, `RTSK.024.1706.01`, `RTSK.024.1707.01`, `RTSK.024.1708.01`, `RTSK.024.1709.01`, `RTSK.024.1710.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)
- **1.13** `RTSK.024.1713.01` ✏️ Top-10 patterns mais citados medidos via `grep -r "FX-.*-NNN" modules/ pkg/ | sort | uniq -c | sort -rn`. *Aceite:* top-10 publicado no README. *Entrega:* seção em `patterns/README.md`. *Escopo:* `patterns/`. *Depende:* `RTSK.024.1712.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.4 — Governança

- **1.14** `RTSK.024.1714.01` ✏️ Pattern-review trimestral (1h) para retirar deprecated + promover úteis. *Aceite:* 1 sessão realizada com notas em `patterns/reviews/`. *Entrega:* notas de review. *Escopo:* `patterns/reviews/`. *Espera:* dono — agenda da sessão. [registro](../registro/README.md)
- **1.15** `RTSK.024.1717.01` ✅ Registrar sintaxe `cognee-revenu:PTN-NNN`, precedência (fundação→ontologia→ADR→padrão→regra→fluxo, cognee-revenu:ADR-019) e catálogo `RFIX` fora da precedência (ledgeros dom. 053, cognee-revenu:O162) em `.base/patterns/index.md`. *Aceite:* `grep` acha os 3 conceitos no `.base/patterns/index.md`. *Entrega:* `.base/patterns/index.md` (pela união dos 2 commits de `docs/padroes-do-contexto`). *Escopo:* `.base/patterns/`. [registro](../registro/ms-024k/2026-11-08-patterns-index-sintaxe.md)
- **1.16** `RTSK.024.1718.01` ✅ Revisar os 27 arquivos trazidos pelo cherry-pick `fd18a75` (commit local `2695b9e`) contra a `develop`. *Aceite:* prova leve verde no sha `c78784f` e sem leitor Go. *Entrega:* `registro/ms-024k/2026-11-08-review-27-arquivos-cherry-pick.md`. *Escopo:* `.base/adrs/`, `.base/flows/`, `.base/rules/`, `.base/plans/00-governance/`, `CLAUDE.md`. [registro](../registro/ms-024k/2026-11-08-review-27-arquivos-cherry-pick.md)
- **1.20** `RTSK.024.1722.01` ✏️ Alinhar ao desenho de seções o resto do catálogo. *Aceite:* README sem catálogo inexistente dado como existente; 1.11–1.14 citam seções e arquivos reais, não `patterns/<área>/`; `TEMPLATE.md` sem «pendente» para o lint. *Entrega:* `patterns/README.md`, `patterns/TEMPLATE.md`, este plano. *Escopo:* `patterns/`, `milestones/backlog/`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1715.01` ✏️ Rodar aceite da MS. *Aceite:* todas as seções das linhas 1.x ✅ e lint verde. *Entrega:* relatório em `registro/ms-024k/`. *Escopo:* `.base/plans/01-architecture/patterns/`. *Espera:* gate — lint patterns. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1716.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `.base/plans/01-architecture/patterns/README.md` — índice atual
- `CLAUDE.md` workflow §2 (TDD-first) consumido por patterns
- ISO 27001 controle 5.37 (documented operating procedures)

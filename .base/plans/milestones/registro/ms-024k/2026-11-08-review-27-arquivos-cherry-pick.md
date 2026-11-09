---
data: 2026-11-08
tarefa: 1.16
estado: ✅
evidencia: 27 arquivos do cherry-pick fd18a75 (commit local 2695b9e, merge c78784f) passam a prova leve na develop; nenhum pacote Go do exchangeos lê .base/ nem CLAUDE.md; N1 leve verde; N2 do head 23fca30 verde contra base d43c6b25 (zero queda nos 11 módulos abaixo do piso 80/70, dívida da MS-024i 1.1); regra «catraca contra base» por decisão do dono via Core em 2026-11-09
pr:
---
Tarefa `RTSK.024.1718.01` (MS-024k 1.16), 2026-11-08.

- **entrega:** revisão dos 27 arquivos trazidos pelo cherry-pick `d1f94934` (upstream `fd18a75` do ramo `chore/padronizacao-plataforma`, preservado como `2695b9e` no `chore/base-padrao-baremetal` e mergeado em `develop` como `c78784f`) contra a `develop`. Prova leve aplicada no sha `c78784f`.
- **conjunto revisado (27 arquivos):**
  - `.base/adrs/` (6): `ADR-009-cambio-de-varejo-como-produto-proprio.md`, `CHANGELOG.md`, `CONVENTIONS.md`, `TEMPLATE.md`, `VERSION`, `index.md` (união).
  - `.base/flows/` (5): `CHANGELOG.md`, `CONVENTIONS.md`, `VERSION`, `index.md` (união), `README.md` removido.
  - `.base/rules/` (14): `CHANGELOG.md`, `CONVENTIONS.md`, `VERSION`, `cfets.md`, `comp.md`, `eod.md`, `gov.md`, `quote.md`, `refdata.md`, `risk.md`, `sett2.md`, `settle.md`, `trade.md`, `index.md` (união).
  - `.base/plans/00-governance/index.md` (1).
  - `CLAUDE.md` (1).
- **o que a revisão substantiva encontrou (comparação arquivo-a-arquivo contra `baremetal/.base/plans/CONVENTIONS.md` § 3.3.5 e os espelhos `treasuryos/.base/{adrs,rules,flows}/`):**
  - `.base/adrs/CONVENTIONS.md` — espelha `baremetal/.base/adrs/` e `onboardos-revenu/.base/adrs/README.md` § «O que exige uma ADR». Coerente com `RULE-008` (doc explica, não decide). **OK.**
  - `.base/adrs/CHANGELOG.md` — abre em `v1.1.0 (2026-10-23)` registrando `ADR-009`; a seção `v1.0.0 (2026-10-22)` fecha-se 1 linha abaixo com as 8 ADRs iniciais. **OK.**
  - `.base/adrs/VERSION` — **débito encontrado: `1.0.0` diverge do CHANGELOG (`v1.1.0`)**. Corrigido neste commit: `VERSION ← 1.1.0`.
  - `.base/adrs/TEMPLATE.md` — define 5 seções (`## Contexto`, `## Decisão`, `## Consequências`, `## Alternativas consideradas`, `## Verificação`) + bloco de convenções do `**Status:** X` em UPPER-CASE do vocabulário fechado. É o template de ADR do baremetal; não se confunde com o `TEMPLATE.md` de `patterns/` (6 seções), que serve a padrões, não a decisões. **OK.**
  - `.base/adrs/ADR-009-...md` — `**Status:** DRAFT — …` no início de linha, formato exigido pelo TEMPLATE.md (linha 4). Alias `ADR-009`, code `RADR.024.009.01`, data `2026-10-23`. 221 linhas. **OK.**
  - `.base/flows/CONVENTIONS.md` — espelha `baremetal/.base/flows/`, `ledgeros/.base/flows/` e `treasuryos/.base/flows/`. Registro de domínios disjunto declarado (`024` + `220`–`249`). **OK.**
  - `.base/flows/CHANGELOG.md` + `.base/flows/VERSION` — ambos em `1.0.0` e sincronizados. **OK.**
  - `.base/flows/README.md` **removido** — delete intencional: o índice passa a ser `index.md`, pela convenção. Nada mais o referencia dentro do módulo. **OK.**
  - `.base/rules/CONVENTIONS.md` — espelha `baremetal/.base/rules/` e `treasuryos/.base/rules/`. Define 10 subdomínios com file + domain assignment (`GOV` 024, `TRADE` 221, `QUOTE` 222, `SETTLE` 223, `CFETS` 224, `SETT2` 225, `REFDATA` 226, `RISK` 227, `COMP` 228, `EOD` 229). **OK.**
  - `.base/rules/CHANGELOG.md` + `.base/rules/VERSION` — ambos em `1.0.0` e sincronizados. **OK.**
  - `.base/rules/{trade,comp,gov,settle,refdata,risk}.md` — cada um abre com `# RN_<AREA> — <título>` e descreve aliases `RN_FX_NNN` com status; estrutura casa com as 11 regras no `index.md`. **OK.**
  - `.base/rules/{quote,sett2,cfets,eod}.md` — abertos com cabeçalho e marcados `Catálogo vazio em 2026-10-22`; são lugares reservados para RN_QUOTE_001, RN_SETT2_001, RN_CFETS_001, RN_EOD_001. **OK** (não há regra a preencher hoje; corpo vazio é intencional, não é débito).
  - `.base/plans/00-governance/index.md` — dif de 54 linhas, apontando a antiga tabela de ADRs para o novo catálogo `.base/adrs/`. **OK.**
  - `CLAUDE.md` — acrescenta um bloco citando `.base/adrs/`, `.base/rules/`, `.base/flows/` depois dos imports `@.claude/context/*` e esclarece a diferença entre `.claude/rules/` (plataforma) e `.base/rules/` (módulo). Coerente com o cherry-pick. **OK.**
- **débitos nomeados contra a convenção da `.base/` do baremetal:**
  1. `.base/adrs/VERSION` estava `1.0.0` e o CHANGELOG abriu `v1.1.0` em 2026-10-23. **Corrigido neste mesmo commit.**
  2. Nenhum outro débito encontrado.
- **autorização:** Core em 2026-11-08: «a fase B está certa, e a união dos 3 índices ficou bem anotada. Nível da 1.16: confirmado, a prova leve que você propôs serve».
- **prova leve (régua do Core para a 1.16, confirmada em 2026-11-08):**
  1. `grep -lE '^(<<<<<<<|=======|>>>>>>>)' <27 arquivos>` — vazio (nenhum marcador de conflito residual).
  2. `gitleaks protect --log-opts "2695b9e^..c78784f"` — `no leaks found`.
  3. `python3 -I baremetal/scripts/registro/conferir-linhas.py .` — total **151 abertas · 0 sem-estado · 0 sem-aceite · 0 sem-entrega · 0 sem-escopo · 0 sem-espera · 0 sem-link · 0 provadas · 0 ☑️ sem N2 · 2 longas**. As 2 longas nascem das próprias tarefas 1.11, 1.12, 1.16 que o fecho da MS-024i e desta linha abriu; o script diz «até o piloto, a § 6 vale por revisão de PR, sem gate» e não é critério de bloqueio.
- **prova «nenhum leitor Go» (Core, 2026-11-08):**
  - `grep -rn '\.base/' --include='*.go' modules/ pkg/ cmd/ internal/` — 1 ocorrência, `modules/trade/domain/doc.go:15`, dentro de comentário `// .base/aasc/ontology/compliance/bacen-cambio-shapes.ttl`. É texto de doc, não leitura.
  - `grep -rn 'CLAUDE\.md' --include='*.go' modules/ pkg/ cmd/ internal/` — vazio.
  - `grep -rn 'embed.*\.base\|embed.*CLAUDE' --include='*.go' .` — vazio (nenhum `//go:embed` aponta pra `.base/` ou `CLAUDE.md`).
  - Conclusão: nenhum pacote Go do exchangeos lê `.base/` nem `CLAUDE.md` em runtime; não há pacote Go a incluir numa prova ampliada desta linha.
- **os três índices** absorvem as 3 frases da HEAD sem violar o shape 1.1.0 (ADR imutável após aceita, fluxo aplica regra/ADR, fundamento de regra é ADR ou padrão). Os arquivos de rules, flows e adrs novos vieram do padrão canônico.
- **remoto:** push/PR/CD esperam o baremetal remoto. Fica tudo em `develop` local, pela regra do dono (v3.427.0).

```prova
n1: c78784f7dca6735a447cafccae9ee84205f126ea verde — grep de marcadores vazio nos 27 arquivos, gitleaks protect no-leaks no diff 2695b9e^..c78784f, conferir-linhas 151/0×6/2 longas, nenhum pacote Go lê `.base/` nem `CLAUDE.md` (`modules/trade/domain/doc.go:15` é comentário) 2026-11-08T21:53:54Z
integracao: develop c78784f7dca6735a447cafccae9ee84205f126ea
n2: 23fca3073c4355816676803f50f3ddba3fc525bb verde — catraca contra base `d43c6b25` passa, zero queda nos 11 módulos abaixo do piso 80/70 (admin/domain 73.0=73.0, cfets_capture/domain 71.2=71.2, cfets_confirmation/domain 60.8=60.8, cls_settlement/domain 75.3=75.3, compliance/domain 67.3=67.3, netreport/domain 72.0=72.0, payin/domain 71.2=71.2, quote/domain 70.2=70.2, refdata/domain 68.3=68.3, trade/application 56.4=56.4, trade/domain 67.2=67.2); diff Go entre base e head é vazio (33 arquivos tocados, 0 `.go`, 0 `go.mod`, 0 `go.sum`); 37/38 passos do gate verdes; piso 80/70 em dívida da MS-024i 1.1; regra «catraca contra base» por decisão do dono via Core em 2026-11-09T04:08:25Z
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

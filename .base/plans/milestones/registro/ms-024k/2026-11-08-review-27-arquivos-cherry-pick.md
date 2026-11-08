---
data: 2026-11-08
tarefa: 1.16
estado: ✅
evidencia: 27 arquivos do cherry-pick fd18a75 (commit local 2695b9e, merge c78784f) passam a prova leve na develop; nenhum pacote Go do exchangeos lê .base/ nem CLAUDE.md
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
- **débitos nomeados contra a convenção da `.base/` do baremetal:** nenhum. Os três índices absorvem as 3 frases da HEAD sem violar o shape 1.1.0 (ADR imutável após aceita, fluxo aplica regra/ADR, fundamento de regra é ADR ou padrão). Os arquivos de rules, flows e adrs novos vieram do padrão canônico.
- **remoto:** push/PR/CD esperam o baremetal remoto. Fica tudo em `develop` local, pela regra do dono (v3.427.0).

```prova
n0: grep -c '<<<<<<<\|=======\|>>>>>>>' nos 27 arquivos do cherry-pick = 0
n1: c78784f7dca6735a447cafccae9ee84205f126ea conferir-linhas 151/0×6/2 + gitleaks no-leaks + sem leitor Go (comentário em modules/trade/domain/doc.go:15 não é leitura) 2026-11-08T21:53:54Z
integracao: develop c78784f7dca6735a447cafccae9ee84205f126ea
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

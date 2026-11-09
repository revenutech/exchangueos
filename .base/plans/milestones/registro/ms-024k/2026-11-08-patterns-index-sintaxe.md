---
data: 2026-11-08
tarefa: 1.15
estado: ✅
evidencia: .base/patterns/index.md contém sintaxe, precedência e RFIX; união dos 2 commits trazidos do ramo docs/padroes-do-contexto; N1 leve verde; N2 do head 23fca30 verde contra base d43c6b25 (zero queda nos 11 módulos abaixo do piso 80/70, dívida da MS-024i 1.1); regra «catraca contra base» por decisão do dono via Core em 2026-11-09
pr:
---
Tarefa `RTSK.024.1717.01` (MS-024k 1.15), 2026-11-08.

- **entrega:** `.base/patterns/index.md` reescrito como união do que a develop tinha (referência a `RADR.091.018.01`, link ao `_template.md`, nota sobre estados via O126/RFLW.091.035) com o conteúdo rico do ramo `docs/padroes-do-contexto` (sintaxe cross-repo `cognee-revenu:PTN-NNN`, precedência `fundação→ontologia→ADR→padrão→regra→fluxo` de cognee-revenu:ADR-019, catálogo `RFIX` fora da precedência de cognee-revenu:O162, matriz de padrões aplicados medida em origin/develop=6d23578 pela MS-018 6.2 da plataforma).
- **origem dos 2 commits trazidos:**
  - `04268cc docs(patterns): .base/patterns/index.md com os PTN da plataforma medidos`
  - `0282e2d docs(patterns): corrige a evidência do .base/patterns/index.md`
- **autorização:** decisão do dono via Core em 2026-11-08, depois da triagem dos 4 ramos não-mergeados: «Trazer à develop».
- **conflito:** add/add em `.base/patterns/index.md`. Resolvido por união, com o ramo como base e as 3 informações da develop preservadas no cabeçalho (`RADR.091.018.01`, `_template.md`, nota O126/RFLW.091.035).
- **prova:** N0. `grep -c "cognee-revenu:PTN-NNN\|ADR-019\|RFIX" .base/patterns/index.md` ≥ 3. Nenhuma `go test` nem `.base/plans/milestones/` reescrita.
- **remoto:** push/PR/CD esperam o baremetal remoto. Fica tudo em `develop` local.

```prova
n1: ad909032b1a91341361a51499cd0b56bc63b2f99 verde — grep do .base/patterns/index.md acha sintaxe, ADR-019 e RFIX; gitleaks protect no-leaks; conferir-linhas sem linha aberta nova
integracao: develop ad909032b1a91341361a51499cd0b56bc63b2f99
n2: 23fca3073c4355816676803f50f3ddba3fc525bb verde — catraca contra base `d43c6b25` passa, zero queda nos 11 módulos abaixo do piso 80/70; diff Go entre base e head vazio (0 `.go`, 0 `go.mod`, 0 `go.sum`); 37/38 passos do gate verdes; piso 80/70 em dívida da MS-024i 1.1; regra «catraca contra base» por decisão do dono via Core em 2026-11-09T04:08:25Z
revisor: n/a — não toca dinheiro, tenant, assinatura nem PII
```

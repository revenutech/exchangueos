# Registro — MS-024n (vuln-fix klauspost/compress)

MS criada em 2026-11-08 por decisão do Core (select do dono) para correção do `GO-2026-6865`. Espelha o upgrade modelo da TreasuryOS `RTSK.280.1307.01` (commit `833b684` em `fix/ms033-compress`).

Sem shape 1.1.0 anterior: a MS nasceu já em CONVENTIONS 1.7.0.

Registros de execução nesta pasta seguem a convenção `YYYY-MM-DD-<slug>-<4hex>.md` do `../README.md`.

## Stack do aceite espelho (fonte: TreasuryOS, 2026-11-08)

Para a 1.1 valer como espelho fiel do modelo, a execução usa a mesma base que a TreasuryOS usou no `833b684`:

- `go version`: `go1.26.10 darwin/arm64`
- `govulncheck`: `v1.9.0`
- DB `govulncheck`: `https://vuln.go.dev`, snapshot de **2026-11-05 23:49:57 UTC**

O `GO-2026-6865` já consta nesse snapshot. Se o DB tiver avançado na hora da execução, achados novos não-`compress` ficam em registro próprio (não contaminam o aceite desta tarefa, que está escopada em um achado).

## Técnica do modelo (congelada pela TreasuryOS em 2026-11-08)

O `833b684` da TreasuryOS é final; conteúdo não muda. A receita literal da 1.1 do exchangeos (corrigida pelo Core em 2026-11-08 — alinhada ao modelo `833b684` e à identityos) é:

- `go get github.com/klauspost/compress@v1.20.2`
- **sem** `go mod tidy` — se o tidy mexer em outra coisa, sai do escopo
- commit único só com `go.mod` + `go.sum`; nenhum outro caminho entra
- diff esperado: linha do `klauspost/compress` em `go.mod` + linhas dele em `go.sum`, nada mais
- `archive/` fica **fora de escopo** (os `go.mod` de `archive/` do modelo não foram tocados; aqui o exchangeos não tem `archive/`, mas a regra é a mesma — só o `go.mod` raiz)
- bateria de 16 passos pela fila deve fechar verde
- `govulncheck ./...` deve passar a «No vulnerabilities found» no recorte `GO-2026-6865`

Lado do modelo: TreasuryOS v1.19.1 → v1.20.2. Lado do exchangeos: v1.17.9 → v1.20.2 (indireto em `go.mod:50`). O salto é maior; isso é esperado.

## Depende satisfeita — 2026-11-08

`treasuryos RTSK 280.1307.01` (MS-033 1.5): merge local feito na develop da TreasuryOS. Evidência:

- Head da develop da TreasuryOS pós-merge: `b09e2156dc0063a22fdf534f54850dc7dd2f5cd9` (sem push)
- Commit modelo permanente: `833b684` (imutável)
- Status na develop deles: build OK + `govulncheck` em «No vulnerabilities found»
- Liberação explícita: «a sua MS-024n pode seguir»

A 1.1 fica agora bloqueada apenas pelo M da Baremetal (destrava a numeração RTSK/RMLS). Estado segue em `✏️` enquanto os códigos forem placeholder.

## M alocado pela Baremetal — 2026-11-08

M=19 confirmado para a MS-024n (§ 3.3.5 da CONVENTIONS). Códigos finais:

- RMLS: `RMLS.024.019.01`
- RTSK (FFF = M×100 + seq):
  - 1.1 → `RTSK.024.1901.01`
  - 2.1 → `RTSK.024.1902.01`
  - 3.1 → `RTSK.024.1903.01`

Baremetal conferiu que M=19 está livre em todos os `.base/` sob `~/projects`, em todos os ramos locais do exchangeos e nos PRs abertos do baremetal. A tabela de M do 024 (024b..024m) já está em `main` do baremetal em **CONVENTIONS 1.9.0**; só a linha da 024n está no ramo de plano local `e268a2f4` (sem push).

Com Depende satisfeita + códigos numerados, a 1.1 passa de `✏️` → `⬜` pela DoR. Execução autorizada pelo Core em 2026-11-08 («a decisão do dono — uma tarefa por módulo, depois do modelo — já cobre isso»). `Espera:` na 1.1 vira «gate — bateria pela fila».

## Execução — 2026-11-08

**Commit do fix:** `1a26cfa` em `fix/ms-024n-klauspost-compress-v1.20.2`. Diff cirúrgico:

- `go.mod`: linha indireta `klauspost/compress v1.17.9 → v1.20.2` (um byte-range).
- `go.sum`: 2 linhas novas (hashes v1.20.2), sem remoção das antigas — esperado sem `go mod tidy`.
- Nada mais tocado. `archive/` fora de escopo (inexistente aqui).

**Receita literal aplicada:**

```
go get github.com/klauspost/compress@v1.20.2
```

**Bateria pela fila (pista sem banco, `REVENU_GATE_SEM_BANCO=1`), comandos go diretos por ausência de `task` no PATH do shell do gate:**

| passo | comando | resultado |
|---|---|---|
| build | `go build ./...` | ok |
| vet | `go vet ./...` | ok |
| test:unit | `go test -race -count=1 -short ./...` | 33 pacotes `ok`, 0 FAIL |
| govulncheck (código) | `govulncheck ./...` | «No vulnerabilities found» |
| govulncheck (código+testes) | `govulncheck -test ./...` | «No vulnerabilities found» |

Log completo: `scratchpad/bateria-ms024n-1a26cfa-v2.log` (sessão).

**Merge local em `develop`:** `422155a` (merge no-ff do ramo `fix/ms-024n-klauspost-compress-v1.20.2`, sem push).

## Reachability — por que o aceite é vacuamente satisfeito

Observação durante o scan: **`GO-2026-6865` não aparece em nenhum dos três níveis do `govulncheck`** no exchangeos — nem como «affects your code», nem como «package you import», nem como «module you require». Antes do bump (v1.17.9) e depois (v1.20.2), com e sem `-test`, o resultado é idêntico:

- Symbol Results: No vulnerabilities found (0 affected).
- Verbose IDs reportadas (não-compress, pré-existentes, fora de escopo desta MS):
  - `GO-2026-6866` (`golang.org/x/net` — inalcançável)
  - `GO-2026-6860` (`golang.org/x/net` — inalcançável)
  - `GO-2026-5932` (`golang.org/x/crypto/openpgp` — inalcançável)

**Hipóteses testadas:**

1. *«v1.17.9 estaria fora da faixa afetada (introduzido depois)»* — **refutada pelo OSV.** `https://vuln.go.dev/ID/GO-2026-6865.json` declara `affected.ranges = [{type: SEMVER, events: [{introduced: 0}, {fixed: 1.20.2}]}]`. Toda versão `<1.20.2` está afetada; v1.17.9 estava na faixa.
2. *«reachability do `govulncheck` não considerou compress como módulo em uso»* — provável. `go mod why` mostra a chain `pkg/outbox/kafka → franz-go/kgo → compress/zstd → compress`, mas o `govulncheck` nem sequer lista o compress como «module you require» pré-bump; a análise de símbolos conclui que o pânico vulnerável não é alcançado em nenhum caminho do exchangeos (código e testes incluídos).

Em consequência: o aceite «`govulncheck ./...` sem `GO-2026-6865`» é satisfeito de forma vacuosa — já era antes do bump. O bump segue em vigor como **higiene/alinhamento** com a decisão do dono (select Core, 2026-11-08: «alinhar todos na v1.20.2»); a MS é uma tarefa válida pela mesma razão que serve à plataforma, não porque resolve um risco alcançável aqui.

**OSV (snapshot):**

```
GO-2026-6865 — Panic on two-block frames in github.com/klauspost/compress
aliases: CVE-2026-116591, GHSA-fpmh-frq8-7rj8
published / modified: 2026-11-05T22:54:32Z
affected.ranges: introduced=0, fixed=1.20.2
```

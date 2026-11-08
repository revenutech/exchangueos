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

O `833b684` da TreasuryOS é final; conteúdo não muda. A receita literal da 1.1 do exchangeos é:

- `go get github.com/klauspost/compress@v1.20.2`
- `go mod tidy`
- commit único só com `go.mod` + `go.sum`; nenhum outro caminho entra
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

M=19 confirmado para a MS-024n (§ 3.3.5 do CONVENTIONS 1.7.0). Códigos finais:

- RMLS: `RMLS.024.019.01`
- RTSK (FFF = M×100 + seq):
  - 1.1 → `RTSK.024.1901.01`
  - 2.1 → `RTSK.024.1902.01`
  - 3.1 → `RTSK.024.1903.01`

Baremetal conferiu que M=19 está livre em todos os `.base/` sob `~/projects`, em todos os ramos locais do exchangeos e nos PRs abertos do baremetal. Registro vive no ramo de plano local da Baremetal `e268a2f4` (sem push).

Com Depende satisfeita + códigos numerados, a 1.1 passa de `✏️` → `⬜` pela DoR. Próximo bloqueio: autorização do dono para rodar o bump real em `go.mod`/`go.sum`. Peer orientação (Core) não substitui — mudança de código de produção exige palavra do dono, mesmo em merge local sem push.

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

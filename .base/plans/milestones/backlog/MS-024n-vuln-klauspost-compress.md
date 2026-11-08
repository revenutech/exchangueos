# MS-024n — Vuln-fix: `klauspost/compress` GO-2026-6865

**Code:** `RMLS.024.019.01` M alocado pela Baremetal em 2026-11-08 (§ 3.3.5 da CONVENTIONS; a linha da 024n está no ramo de plano da Baremetal `e268a2f4`, sem push — a tabela 024b..m já está em `main` em CONVENTIONS 1.9.0).
**Status:** BACKLOG
**Owner:** Platform
**Origem:** criada em 2026-11-08 por decisão do Core (select do dono); espelha o upgrade modelo da TreasuryOS MS-033 1.5 (`RTSK.280.1307.01`, commit `833b684` em `fix/ms033-compress`). Sem shape 1.1.0 anterior.

## Por quê

Vulnerabilidade `GO-2026-6865` — pânico em `github.com/klauspost/compress`, corrigido na v1.20.2. O `go.mod` do exchangeos declara `klauspost/compress v1.17.9 // indirect` (linha 50), puxado via dependências transitivas. A decisão do dono (select via Core, 08/11) é que cada módulo faz a correção localmente; a TreasuryOS vai primeiro e serve de modelo.

## Descrição

Subir `github.com/klauspost/compress` de v1.17.9 para v1.20.2 em um único commit isolado (só `go.mod` + `go.sum`), merge local sem push (regra v3.427.0). Aceite pelo `govulncheck ./...` limpo do achado + bateria pela fila verde no head do commit.

## Fase 1 — Implementação

### Etapa 1.1 — Upgrade

- **1.1** `RTSK.024.1901.01` ✅ Subir `github.com/klauspost/compress` de v1.17.9 para v1.20.2 em commit isolado (só `go.mod` + `go.sum`). *Aceite:* `govulncheck ./...` sem `GO-2026-6865` (satisfeito vacuamente — ausência de reachability; ver registro) + bateria pela fila verde (33 pacotes ok). *Entrega:* `1a26cfa` em `fix/ms-024n-klauspost-compress-v1.20.2`, mergeado em `develop` local em `422155a` (sem push). *Escopo:* `go.mod`, `go.sum`. *Depende:* `treasuryos RTSK 280.1307.01` — satisfeita em 2026-11-08. *Espera:* —. [registro](../registro/ms-024n/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1902.01` ✅ Rodar aceite da MS. *Aceite:* 1.1 verde + `govulncheck ./...` limpo do achado + bateria pela fila verde. *Entrega:* evidências pré/pós e OSV registradas em `registro/ms-024n/README.md`. *Escopo:* módulo inteiro. *Depende:* 1.1 — satisfeita. *Espera:* —. [registro](../registro/ms-024n/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1903.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Depende:* 2.1. *Espera:* dono — aprovação final. [registro](../registro/ms-024n/README.md)

## Referências

- `go.mod:50` — `github.com/klauspost/compress v1.17.9 // indirect`
- Modelo: TreasuryOS `RTSK.280.1307.01`, commit `833b684` em `fix/ms033-compress`
- CONVENTIONS 1.7.0 § 3.3.5 (M por sub-letra do DDD) e § 6 (forma de dependência inter-repositório)
- Regra v3.427.0 (ramo por MS, merge local sem push)
- GO-2026-6865 (govulncheck)

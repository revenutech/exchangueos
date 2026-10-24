# Catálogo de Fluxos — exchangeos

Fluxos `RFLW.DDD.FFF.VV`. Um fluxo aplica regras e ADRs. Convenções em
[`CONVENTIONS.md`](CONVENTIONS.md). Regras referenciadas:
[`../rules/`](../rules/index.md). Decisões: [`../adrs/`](../adrs/index.md).
Versão: `VERSION`.

## Registros

**Medido em 2026-10-22:** 8 fluxos existentes com o código legado
`RFLW.024.*` (mantidos pela regra de zero-renumeração da § 3.3.5), todos
`DRAFT` — nenhum ambiente do exchangeos está operando. Novos fluxos
nascem no subdomínio da matéria (`221`–`249`), não em `024`.

| Código | Fluxo | Subdomínio | Status |
|--------|-------|------------|--------|
| [RFLW.024.001.01](trade/RFLW.024.001.01.md) | Book FX Spot Trade via CLS (EUR/USD) | trade (221) | DRAFT |
| [RFLW.024.002.01](trade/RFLW.024.002.01.md) | Settle FX Trade via CLS PvP | trade (221) | DRAFT |
| [RFLW.024.010.01](quote/RFLW.024.010.01.md) | RFQ Streaming Lifecycle (Requested → Quoted → Accepted) | quote (222) | DRAFT |
| [RFLW.024.020.01](cls_settlement/RFLW.024.020.01.md) | CLS Daily Cycle Lifecycle (07:00 Open → 12:00 Close) | cls_settlement (223) | DRAFT |
| [RFLW.024.030.01](compliance/RFLW.024.030.01.md) | BACEN Classification + IOF Computation on Trade Booked | compliance (228) | DRAFT |
| [RFLW.024.040.01](cfets/RFLW.024.040.01.md) | CFETS Trade Capture (fxtr.031 → fxtr.032 → fxtr.033) | cfets (224) | DRAFT |
| [RFLW.024.050.01](eod/RFLW.024.050.01.md) | End-of-Day Batch (PTAX → MTM → Position Snapshot → BACEN Report) | eod (229) | DRAFT |
| [RFLW.024.060.01](risk/RFLW.024.060.01.md) | Pre-Trade Risk Limit Check + Reserve | risk (227) | DRAFT |

**Próximos** (um por subdomínio da faixa nova):
`RFLW.221.001` · `RFLW.222.001` · `RFLW.223.001` · `RFLW.224.001` ·
`RFLW.225.001` · `RFLW.226.001` · `RFLW.227.001` · `RFLW.228.001` ·
`RFLW.229.001`.

## Achados do catálogo (medidos 2026-10-22)

1. **O README.md anterior anunciava «85 fluxos agendados nas 6 subpastas».**
   Medido: existem **8** fluxos, em **7** subpastas (`trade`, `quote`,
   `cls_settlement`, `cfets`, `compliance`, `eod`, `risk`). O número 85 era
   intenção, não realidade — ficou neste índice como achado histórico, não
   como meta.
2. **Todos `DRAFT`.** Nenhum ambiente operando; nenhum fluxo tem evidência
   de execução real. Promover a `ACTIVE` depende de ambiente, não só de
   código verde em teste.
3. **Formato misto.** Os 8 fluxos usam YAML *front matter*
   (`Code: RFLW.024.NNN.NN`); fluxos novos nascem com cabeçalho em tabela
   (`| **Code** | RFLW.… |`), como mandam as `CONVENTIONS.md` da plataforma.

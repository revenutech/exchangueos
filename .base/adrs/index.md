# Architecture Decision Records — exchangeos

Decisões de arquitetura do exchangeos. Cada ADR é **imutável depois de
aceita**: mudança de rumo é ADR nova, que supersede a anterior, e as duas
atualizam o status. **Um alias `ADR-NNN` nunca se reutiliza.** Convenções
em [`CONVENTIONS.md`](CONVENTIONS.md). Template: [TEMPLATE.md](TEMPLATE.md).

O código canônico `RADR.DDD.FFF.VV` usa o domínio do registro único da
plataforma (`baremetal/.base/plans/CONVENTIONS.md` § 3.3.5, faixa
**`220`–`249`**). O número (`ADR-NNN`) e o `FFF` são alocados **aqui**, no
mesmo commit do documento. Não existem números reservados.

## Registros

> **Nenhuma ADR deste catálogo está `ACCEPTED` ainda.** As decisões abaixo
> estão **no código** (CLAUDE.md, go.mod, lefthook, Taskfile), e o documento
> `.base/adrs/*.md` que lavra cada uma está por escrever. O status `DRAFT`
> aqui é literal: o alvo canônico ainda não existe.

| ID | Code | Título | Status | Data |
|----|------|--------|--------|------|
| ADR-001 | `RADR.024.001.01` | Hub CockroachDB compartilhado, TLS verify-full desde o dia 1 | DRAFT — decisão no código, documento por escrever | 2026-10-22 |
| ADR-002 | `RADR.024.002.01` | ISO 20022 `fxtr` é o único na borda; `fxti` e `fxmt` não existem; *quote* e *amendment* vivem como gRPC interno | DRAFT — idem | 2026-10-22 |
| ADR-003 | `RADR.024.003.01` | `decimal.Decimal` (shopspring) para *money* e *rate*; `float64` proibido | DRAFT — idem | 2026-10-22 |
| ADR-004 | `RADR.024.004.01` | Portas `8084` HTTP / `9084` gRPC; realocado por colisão com `riskos` | DRAFT — idem | 2026-10-22 |
| ADR-005 | `RADR.024.005.01` | Lefthook em três camadas (pre-commit < 30s · pre-push < 3 min · pre-merge < 15 min); `--no-verify` bloqueado por *wrapper* | DRAFT — idem | 2026-10-22 |
| ADR-006 | `RADR.024.006.01` | Consumo do `revenu-common` no lugar das cópias locais de identificadores e eventos canônicos | DRAFT — idem | 2026-10-22 |
| ADR-007 | `RADR.024.007.01` | *Dual-ledger*: `ledgeros` como autoridade contábil, Temenos como alvo de produção, *stub* fora de produção | DRAFT — idem | 2026-10-22 |
| ADR-008 | `RADR.024.008.01` | `exchangeos` é módulo próprio, não *embedded* em `paymentos` nem em `ledgeros` | DRAFT — idem | 2026-10-22 |
| ADR-009 | `RADR.024.009.01` | [ExchangeOS passa a ter dois produtos coexistentes — interbancário (atual) e câmbio de varejo (novo, originado pelo app)](ADR-009-cambio-de-varejo-como-produto-proprio.md) | DRAFT — conteúdo decidido pelo dono em 2026-10-23 (contraparte, enquadramento, pares, limites, eventos, rotas); aguarda revisão final antes de ir a `PROPOSED` | 2026-10-23 |

**Próximo:** `ADR-010` · `RADR.024.010.01` (ADRs transversais ficam em `024`;
decisões de subdomínio nascem no FFF da matéria — 221 *trade*, 222 *quote*,
223 CLS, 224 CFETS, 225 *settlement*, 226 *refdata*, 227 risco,
228 *compliance*, 229 EOD — começando no `001` da sua faixa).

## Decisões em doc que **não** viraram ADR

A seção «Decision Records (ADRs)» de
[`.base/plans/00-governance/index.md`](../plans/00-governance/index.md) tinha 8
itens. O critério (`CONVENTIONS.md` § 3) pede **as quatro** condições:
alternativa real, porta que custa reabrir, decisão que atravessa contextos
e porquê que o código não mostra. Nem todo item passa.

| Item do `00-governance/index.md` | Destino | Por quê |
|---|---|---|
| ADR-001 *Microservice Architecture* | **sem ADR** | não é decisão deste módulo; é da plataforma. Fica no índice da plataforma, se existir |
| ADR-009 *AccountOS Architecture* | **sem ADR** | é do `accountos`, não do `exchangeos` — doc do módulo errado |
| ADR-010 *OnboardOS Architecture* | **sem ADR** | é do `onboardos-revenu`, não do `exchangeos` — doc do módulo errado |
| ADR-011 *ExchangeOS Architecture* (standalone, multi-CCY dual-ledger, CLS, SWIFT FIN vs MX) | virou **ADR-007** e **ADR-008**, divididas | o item original misturava duas decisões independentes (*dual-ledger* e *standalone vs embedded*); separar segue `RULE-008` |
| ADR-012 *PvP Strategy* (CLS first vs *gross* com Herstatt) | **candidata** a nova ADR em `223` | passa nas quatro; esperando o dono decidir se CLS *first* é mesmo política ou dedução do contrato |
| ADR-013 *ISO 20022 FX Coverage* | virou **ADR-002** | mesma matéria, nome mais honesto (o original falava em «fxti/fxmt obsolete», mas nunca existiram) |
| ADR-014 *Dual-Ledger Strategy* | virou **ADR-007** | — |
| ADR-015 *Shared CRDB Hub* | virou **ADR-001** | — |

A tabela em `00-governance/index.md` deixa de listar ADRs: passa a apontar
para este catálogo. A renumeração interna (`ADR-001` daqui ≠ `ADR-001` da
tabela antiga) é consequência de **o alias nunca ter sido alocado em
índice** — a tabela antiga era planejamento, não registro.

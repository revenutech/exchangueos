# 00 — Governance

> **Workstream:** Governance
> **Versao:** 1.0.0
> **Status:** DRAFT

## Conteudo

| Documento | Status | Descricao |
|-----------|--------|-----------|
| `program-charter.md` | TODO | Visao + escopo + principios + governance model |
| `glossary.md` | TODO | Ubiquitous Language (FX domain + FIBO + ISO 20022) |
| `quality-gates.md` | TODO | 6 gates GS-1..GS-6 + Allenty Activate pipeline |
| `risk-register.md` | TODO | 90+ riscos identificados + mitigacoes |
| `open-questions.md` | TODO | 108 open questions categorizadas |
| `cross-references.md` | TODO | 200+ fontes oficiais consultadas |
| `integration-audit.md` | TODO | Matrix 4 vetores × 13 modulos (Kafka/DB/gRPC/Sync) — §20 monolitico |
| `enterprise-benchmark.md` | TODO | Allenty Triad (Oracle x SAP x Apple) — 12 principios |
| `document-framework.md` | TODO | Padrao de estruturacao de docs do workstream |
| `decision-records/` | TODO | ADRs (ADR-001..050+) |
| `audits/` | TODO | Audit reports |
| `legal/` | TODO | Regulatory + contracts |
| `product/` | TODO | Product roadmap context |
| `standards/` | TODO | Internal standards |

## Decision Records (ADRs)

**O catálogo canônico de ADRs do exchangeos vive em
[`../../adrs/`](../../adrs/index.md)**, no padrão unificado da plataforma
(`baremetal/.base/plans/CONVENTIONS.md` § 3.3.5, faixa `024` + `220`–`249`).
Esta seção deixa de listar ADRs — doc explica, não decide (`RULE-008`).

A tabela anterior listava 8 itens, dos quais 5 passam no critério de ADR
(`alternativa real · porta cara de reabrir · atravessa contextos · porquê
não dedutível do código`) e 3 não (`ADR-001`, `ADR-009` e `ADR-010` eram
de outros módulos). A migração está em `../../adrs/index.md` § «Decisões
em doc que não viraram ADR».

## Sources

- §1 (Visao Geral) + §5 (Milestones) + §6 (Cross-References) + §7 (Riscos) + §9 (Open Questions) + §20 (Integration Audit) do [snapshot monolitico v3.11.7](../_archive/allenty-v3.11.7-monolithic-plan.md)
- Pattern de referencia: [LedgerOS 00-governance](../../../../ledgeros/.base/plans/00-governance/)

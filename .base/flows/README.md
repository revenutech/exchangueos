# ExchangeOS Flows Suite

Mermaid sequence + flowchart diagrams for every canonical FX flow.

## Naming convention

```
RFLW.024.NNN.NN
       │   │  │
       │   │  └── revision (01, 02, …)
       │   └────── flow sequence within domain
       └────────── Allenty domain (024 = ExchangeOS)
```

## Layout

```
.base/flows/
├── README.md               # this file
├── trade/                  # 12 flows (book/confirm/amend/cancel/settle paths)
│   ├── RFLW.024.001.01.md  ✅ representative (Book FX Spot via CLS)
│   └── ...
├── quote/                  # 8 flows
├── cls_settlement/         # 15 flows
├── cfets/                  # 10 flows (capture + confirmation per PTPP)
├── compliance/             # 12 flows (DEC, IOF, SISCOAF, sanctions)
└── eod/                    # 6 flows (PTAX → MTM → snapshot → BACEN report)
```

## Required structure per file

Each `.md` carries:

1. **YAML metadata header** — Code, Domain, Module, Version, Status, Title, Traceability
2. **Description** — one short paragraph
3. **Pre-conditions**
4. **Actors / Participants**
5. **Mermaid sequence diagram** — happy path
6. **Mermaid flowchart** — error paths
7. **Business Rules Applied** — table of RN_FX_* codes
8. **Observability** — OTel spans + metrics + logs
9. **Compliance Notes** — BACEN/SISCOAF hooks if applicable
10. **Related Patterns** — pointers to FX-* catalogs

Predecessor + Successor links keep flows navigable.

## Status

- ✅ trade/RFLW.024.001.01 (Book FX Spot via CLS) — representative
- ⏳ 84 remaining flows scheduled across the 6 sub-folders

> **Nota de frescor 2026-09-17 (auditoria).** O bloco Layout acima omite a subpasta `risk/`, que existe (`risk/RFLW.024.060.01.md`). Incluí-la no layout.

---
data: 2026-11-10
tarefa: 1.1
estado: ✅
evidencia: interface `Submitter` em `pkg/bacen/submission/`, com redação do Body; revisada pela IdentityOS; N1 verde, na develop em 208dda3; N2 do lote 9.1 verde em 88938b8
pr:
---
Tarefa `RTSK.024.1301.01` (MS-024f 1.1), 2026-11-10, lote 9.1 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `pkg/bacen/submission/submitter.go`, `submitter_test.go` e `redact_test.go`. `Submit(ctx, Payload) (Receipt, error)` e `Query(ctx, protocolID) (Receipt, error)`; tipos próprios `Kind` (DEC, SCE-IED, SCE-CREDITO, SCE-CBE), `Payload`, `Status` e `Receipt`.
- **revisão:** IdentityOS aprovou `973249c` com recomendação forte; aplicada em `831cfdb`: `String()`/`LogValue()` só com Kind, ReferenceID e `body_len`; doc do Body (byte a byte, XAdES), da `RejectionReason` (pode ter PII) e do `Submit` (idempotente pela ReferenceID). Aprovado em `831cfdb`.
- **débito (não bloqueia):** falta `GoString()` (o `%#v` imprime o Body) e `json:"-"` no Body se o struct for serializado. Vira tarefa de seguimento.
- **nota para a MS-024f 1.10:** o `ReportType` de `modules/compliance/domain` é SISBACEN/CCS/CAMBIO, não DEC/SCE; o pacote não importa o domínio.
- **N1:** pista sem banco, build, vet, `test -race ./...` (34 pacotes) e `gitleaks` verdes.

```prova
n1: 831cfdb72c936da53e655cdc168c2e9c8041919c rodada exchangeos-submitter@2026-11-10T15:07:47Z
n2: 88938b8ccca11c4f3200dc12c135a54a9c252167 rodada exchangeos-n2-lote91@2026-11-10T15:46:08Z — build, vet, test -race ./... (35 pacotes), govulncheck, gitleaks 663fdaf..head, escopo 8/8, lint-patterns e catraca head × base 663fdaf sem queda (33 → 35 pacotes)
integracao: develop 208dda36142ba8df17911c29a0435056f7f4de9d
revisor: sessão IdentityOS — aprovado 831cfdb em 2026-11-10 (assinatura, PII)
```

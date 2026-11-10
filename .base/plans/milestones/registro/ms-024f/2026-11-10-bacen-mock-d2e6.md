---
data: 2026-11-10
tarefa: 1.14
estado: ✅
evidencia: mock do gateway BACEN em `tests/integration/` (DEC e SCE-*), com teste próprio; N1 verde, na develop em 48eded1; N2 do lote 9.1 verde em 88938b8
pr:
---
Tarefa `RTSK.024.1314.01` (MS-024f 1.14), 2026-11-10, lote 9.1 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `tests/integration/bacen_mock_server.go` e `bacen_mock_server_test.go`, tag `integration`. `NewBACENMock(t)`, `URL()` e `RejectReference(ref, reason)`; POST devolve 202 RECEIVED, idempotente pela referência; o 1º GET passa a ACCEPTED ou REJECTED; protocolo desconhecido dá 404.
- **N1:** pista sem banco, `go build ./...`, `go vet ./...`, `go test -race ./...`, `go vet -tags integration ./tests/integration/`, `go test -race -tags integration -run BACENMock ./tests/integration/` e `gitleaks` verdes. O `-run` é necessário: os outros testes do pacote chamam o servidor vivo em `:8084`.
- **fora do escopo:** `tests/integration/admin_crud_test.go`, já na develop, não passa no `gofmt`.

```prova
n1: 7b6587f7ce906941a466069563059571eb3d0aa7 rodada exchangeos-bacen-mock@2026-11-10T14:46:33Z
n2: 88938b8ccca11c4f3200dc12c135a54a9c252167 rodada exchangeos-n2-lote91@2026-11-10T15:46:08Z — build, vet, test -race ./... (35 pacotes), govulncheck, gitleaks 663fdaf..head, escopo 8/8, lint-patterns e catraca head × base 663fdaf sem queda (33 → 35 pacotes)
integracao: develop 48eded1c9e9569e6853768d715c6dafb148e5c76
revisor: n/a — mock de teste, sem dado real; não toca dinheiro, tenant, assinatura nem PII
```

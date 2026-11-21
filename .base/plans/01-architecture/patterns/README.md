# ExchangeOS Pattern Catalog

21 catalogs totalling 865 patterns. Patterns are cited inline in code via
`// FX-<CATALOG>-NNN` comments.

## Catalogs

| File | Catalog | Count | Status |
|------|---------|-------|--------|
| 200-fx-golang-patterns.md          | FX-GP-* (Go)                          | 40 | ✅ 5 docs |
| 201-fx-ddd-patterns.md             | FX-DDD-* (Domain-Driven Design)        | 35 | ✅ 4 docs |
| 202-fx-eda-patterns.md             | FX-EDA-* (Event-Driven Architecture)   | 45 | ✅ 4 docs |
| 204-fx-compliance-patterns.md      | FX-CMP-* (Compliance)                  | 15 | ⏳ |
| 205-fx-cockroachdb-patterns.md     | FX-CP-* (CockroachDB)                  | 50 | ✅ 5 docs |
| 206-fx-kafka-patterns.md           | FX-KP-* (Kafka)                        | 60 | ✅ 4 docs |
| 207-fx-flink-patterns.md           | FX-FP-* (Apache Flink)                 | 40 | ⏳ |
| 210-fx-devsecops-cicd-patterns.md  | FX-DS-* (DevSecOps + CI-CD)            | 50 | ✅ 5 docs |
| 211-fx-kubernetes-patterns.md      | FX-K8S-*                                | 40 | ⏳ |
| 212-fx-terraform-gcp-patterns.md   | FX-IAC-*                                | 40 | ⏳ |
| 213-fx-docker-container-patterns.md| FX-DOC-*                                | 20 | ⏳ |
| 220-fx-grpc-patterns.md            | FX-GRPC-*                               | 55 | ⏳ |
| 221-fx-api-rest-patterns.md        | FX-API-*                                | 50 | ⏳ |
| 222-fx-asyncapi-patterns.md        | FX-ASYNC-*                              | 45 | ⏳ |
| 230-fx-iam-rbac-patterns.md        | FX-IAM-*                                | 50 | ⏳ |
| 240-fx-opentelemetry-patterns.md   | FX-OTEL-*                               | 60 | ⏳ |
| 250-fx-testing-patterns.md         | FX-TEST-*                               | 40 | ⏳ |
| 260-fx-qa-tdd-e2e-patterns.md      | FX-QA-*                                 | 35 | ⏳ |
| 270-fx-sync-cross-module-patterns.md| FX-SYNC-*                              | 40 | ⏳ |
| 280-fx-integration-verification-patterns.md | FX-INT-*                       | 25 | ⏳ |
| 290-fx-cross-platform-patterns.md  | FX-XOS-*                                | 20 | ⏳ |
| 300-fx-precommit-enforcement-patterns.md | FX-COMMIT-*                       | 25 | ⏳ |
| **Total** | — | **865** | 3 representative ✅ |

## Pattern template

Each pattern follows the same shape:

```
### FX-<CAT>-NNN — <title>
Context: when does this apply?
Problem: what does it solve?
Solution: the canonical shape
Example: snippet or file pointer in the repo
Anti-pattern: what NOT to do
Related: other FX-* patterns
```

## Milestones

- **MS-023m** (App layer: FX-GP/DDD/EDA) — representative 200-fx-golang-patterns.md ✅
- **MS-023n** (Infra: FX-CP/KP/FP) — representative 205-fx-cockroachdb-patterns.md ✅
- **MS-023o** (DevSecOps+IaC: FX-DS/K8S/IAC/DOC) — representative 210-fx-devsecops-cicd-patterns.md ✅

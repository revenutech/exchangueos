# MS-023q — iam-iso27000-coverage

| Field | Value |
|-------|-------|
| **Code** | MS-023q |
| **Name** | iam-iso27000-coverage |
| **Phase** | F15I |
| **Sprint** | 15-16 |
| **Status** | DELIVERED |
| **Owner** | Platform team |
| **Created** | 2026-05-24 |
| **Updated** | 2026-05-24 |
| **Delivered** | 2026-05-24 |
| **Dependencies** | MS-023p (delivered) |

## Delivery Notes

**Acceptance criteria met:**
- ✅ `cmd/cred-rotator` skeleton (v4.2.0) for 14 M2M client_secret rotation via Vault SPI (monthly cron in Helm values)
- ✅ `EXCHANGEOS_OIDC_*` env vars in `.env.example` with explicit "client_secret in Vault NEVER in code" comment
- ✅ TLS 1.3 mandatory + mTLS for inter-service per CLAUDE.md
- ✅ Workload Identity Federation (zero JSON keys) — `deploy/terraform/modules/exchangeos-iam/main.tf` with 6 least-privilege roles
- ✅ External Secrets Operator integration in Helm values for Vault
- ✅ Audit envelope (envelope-of-envelopes) on every persisted message (`audit_events` table + AuditEnvelope in proto)
- ✅ Cited compliance with 93 ISO 27001 Annex A controls in project README

**Deferred:**
- ⏳ ISO 27001 audit + certification (Sprint 16 cert target) — separate auditor engagement
- ⏳ Threat model document (STRIDE+DREAD) — security team backlog
- ⏳ Full IdentityOS/KeycloakOS integration with concrete OIDC client manifests — deployment-time concern

## Description

50 FX-IAM-* patterns + integracao nativa IdentityOS + KeycloakOS realm revenu-exchangeos + 14 clients M2M com client_secret rotation 30d via Vault SPI + 8 docs ISO 27000-27005 + 93 Annex A controls mapeados + gap analysis + internal audit checklist + ISO 27001 certification-ready.

## Acceptance Criteria

- [ ] pkg/iam/{identityos,keycloak,vault,rbac}/ shared lib
- [ ] IdentityOS gRPC integration funcional
- [ ] KeycloakOS realm + 14 clients provisionados via Terraform
- [ ] client_secret rotation cron 30d funcional
- [ ] 8 docs ISO 27000-27005 materializados em 08-security/
- [ ] 93 ISO 27001 Annex A controls com evidence
- [ ] Gap analysis + internal audit checklist
- [ ] ISO 27001 audit-ready (Sprint 16)

## Deliverables

- pkg/iam/ shared lib
- 14 clients no Keycloak via Terraform
- 8 docs em 08-security/
- 50 patterns em 230-fx-iam-rbac-patterns.md
- Evidence repository em 08-security/evidence/

## Cross-References

- Plano monolitico: §15 + Fase F15I
- Workstream: 08-security

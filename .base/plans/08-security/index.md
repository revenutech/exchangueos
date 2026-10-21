# 08 — Security

> **Workstream:** Security
> **Versao:** 1.0.0
> **Status:** DRAFT — ISO 27001 certification target Sprint 16

## Conteudo

| Documento | Status | Descricao |
|-----------|--------|-----------|
| `iam-iso27000-coverage.md` | TODO | IAM Integration (IdentityOS + KeycloakOS) + ISO 27000-27005 (§15 monolitico) |
| `iso27000-fx-framework.md` | TODO | ISO/IEC 27000:2022 framework adoption + 6-layer stack + glossary |
| `iso27001-fx-annex-a-mapping.md` | TODO | **93 ISO 27001:2022 Annex A controls** mapeados para ExchangeOS + evidence per control |
| `iso27002-fx-controls-implementation.md` | TODO | ISO/IEC 27002:2022 implementation guidance + Go/Helm/Terraform snippets |
| `iso27003-fx-isms-roadmap.md` | TODO | ISO/IEC 27003:2017 ISMS 4-phase roadmap (Foundation → Domain → Integration → Certification) |
| `iso27004-fx-security-metrics.md` | TODO | ISO/IEC 27004:2016 — 10+ security metrics SLI/SLO + Prometheus exporters |
| `iso27005-fx-risk-management.md` | TODO | ISO/IEC 27005:2022 — STRIDE+DREAD threat model + risk register FX-specific |
| `iam-scope-catalog-fx.md` | TODO | Scope catalog `exchangeos:<resource>:<verb>` + ABAC rules + 5 roles |
| `oauth2-fx-zero-trust-plan.md` | TODO | OAuth2 client_credentials zero-trust + token policies + revocation |
| `threat-model.md` | TODO | STRIDE threat model |
| `threat-model-supply-chain.md` | TODO | SLSA threat model supply chain |
| `cryptography.md` | TODO | Cryptography standards (TLS 1.3 + AES-256-GCM + RS256 + KMS HSM) |
| `vault-pki-rotation.md` | TODO | Vault PKI cert rotation 90d playbook |
| `gcp-vpc-service-controls.md` | TODO | VPC SC perimeter security |
| `binary-authorization-policy.md` | TODO | Binary Authorization GKE policy + break-glass procedure |
| `evidence/` | TODO | ISO 27001 evidence repository |
| `policies/` | TODO | Security policies |
| `templates/` | TODO | Security templates |

## ISO 27000 Series Coverage

| Standard | Title | Target |
|----------|-------|--------|
| ISO/IEC 27000:2022 | Overview & vocabulary | Glossary aligned with LedgerOS |
| ISO/IEC 27001:2022 | ISMS Requirements | **Certification target Sprint 16** — 93 Annex A controls |
| ISO/IEC 27002:2022 | Information security controls | Implementation guidance per control |
| ISO/IEC 27003:2017 | ISMS implementation guidance | 4-phase roadmap |
| ISO/IEC 27004:2016 | Monitoring + measurement | Security metrics SLIs/SLOs |
| ISO/IEC 27005:2022 | Risk management | STRIDE + DREAD integration |

## IAM Stack

- **IdentityOS** (`:8084 HTTP / :9084 gRPC`) — AuthZ Policy + Sessions + Consent + Federation
- **KeycloakOS** (v26.5.3) — Realm `revenu-exchangeos` + Organizations multi-tenancy + 14 clients M2M + 2 user clients
- **Vault** (HCP managed) — client_secret rotation 30d + PKI cert 90d + KMS HSM
- **KrakenD** — JWT validation no edge + JWKS cache 5min + Bloom filter revocation

## Sources

- §15 (IAM Integration + ISO 27000-27005) do [snapshot monolitico v3.11.7](../_archive/allenty-v3.11.7-monolithic-plan.md)
- Pattern de referencia: [LedgerOS 08-security](../../../../ledgeros/.base/plans/08-security/)

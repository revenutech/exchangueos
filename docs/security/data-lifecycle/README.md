# Data Lifecycle + LGPD — ExchangeOS

> Per-table retention schedule, archival policy, right-to-erasure workflow, and
> data classification matrix. Required for LGPD (Lei 13.709/2018) compliance +
> ISO 27001 controls 8.10 (information deletion), 8.11 (data masking), 8.12
> (data leakage prevention).

## Data classification matrix

| Class | Definition | Examples in ExchangeOS | Encryption | Retention default |
|-------|-----------|----------------------|------------|-------------------|
| **C1 — Public** | No risk if disclosed | currency codes, calendars (refdata) | TLS in transit only | Indefinite |
| **C2 — Internal** | Operational; minor harm | trade IDs, BIC codes, system events | TLS + at-rest GCP default | 7 years |
| **C3 — Confidential** | Commercial harm if leaked | trade prices, NOP, P&L, BACEN report payloads | TLS 1.3 + CMEK HSM | 7 years (regulatory) |
| **C4 — Restricted (PII)** | LGPD-regulated; legal harm | counterparty actor names, beneficial owner data, screening matches | TLS 1.3 + CMEK HSM + per-field column encryption (planned) | 5 years post-relationship; right-to-erasure on request |

LGPD legal basis used per dataset is recorded in this table — most ExchangeOS
data falls under "execução de contrato" (Art. 7º V) or "obrigação legal"
(Art. 7º II, BACEN/COAF reports).

## Retention schedule per table

> Source: cross-referenced against `migrations/00000N_*.up.sql` + LGPD Art. 16
> + BACEN Resolução BCB 119 (record-keeping 5+ years).

| Table | Class | Legal basis | Retention | Archival → GCS Coldline at | Right-to-erasure scope |
|-------|-------|-------------|-----------|----------------------------|------------------------|
| `tenants` | C2 | Contract | 7 years post-termination | T+1y | Soft-delete on request; anonymise PII after legal hold |
| `actors` | C4 | Contract + legal | 5 years post-relationship | T+1y | Erasure eligible after BACEN 5y window |
| `audit_events` | C3 | Legal (Art. 7º II) | 10 years | T+1y | NEVER (regulatory hold) |
| `counterparties` | C4 | Contract | 7 years post last trade | T+1y | Anonymise BIC + name; keep aggregate stats |
| `fx_trades` | C3 | Legal (BACEN) | 10 years | T+1y | NEVER (regulatory hold) |
| `trade_amendments` | C3 | Legal | 10 years | T+1y | NEVER |
| `cls_cycles` + `cls_cycle_trades` | C3 | Legal (CLS audit) | 10 years | T+1y | NEVER |
| `payin_instructions` | C3 | Legal | 10 years | T+1y | NEVER |
| `net_reports` | C3 | Legal | 10 years | T+1y | NEVER |
| `quotes` + `rfqs` | C3 | Contract | 5 years | T+90d | Erasable (no regulatory hold) |
| `quote_streams` | C2 | Contract | 1 year | T+30d | Erasable |
| `currencies` / `calendars` / `currency_pairs` / `netting_cutoffs` | C1 | — | Indefinite | Never | N/A |
| `bic_records` | C2 | — | Indefinite | Never | N/A (public registry data) |
| `ssis` | C4 | Contract | 7 years post-deactivation | T+1y | Anonymise account number + IBAN |
| `classifications` | C3 | Legal (BACEN Circ 3.690) | 10 years | T+1y | NEVER |
| `iof_computations` | C3 | Legal (Decreto 12.499/2025) | 10 years | T+1y | NEVER |
| `bacen_reports` | C3 | Legal | 10 years | T+1y | NEVER |
| `screening_results` | C4 | Legal + legitimate interest | 7 years | T+1y | Anonymise hit details after window |
| `system_events` | C3 | Legal (CLS admi) | 7 years | T+1y | NEVER |
| `eod_jobs` | C2 | Operational | 1 year | T+90d | N/A |
| `risk_limits` | C3 | Operational | 5 years | T+1y | N/A |
| `positions` | C3 | Legal | 10 years | T+1y | NEVER |
| `outbox_events` + `_archive` | C3 | Audit | 90 days hot / 7 years cold | T+90d | NEVER (audit) |

## Archival cron (planned)

`cmd/archiver/main.go` — nightly job (02:00 SP) that:

1. For each table with archival rule, selects rows older than the threshold
2. Bulk-exports to GCS `gs://exchangeos-archive-<env>/<table>/YYYY/MM/DD/<batch>.parquet.gz`
3. Computes SHA256 + writes to manifest table `archive_manifest`
4. Marks rows as archived (sets `archived_at`)
5. After 30 days of `archived_at`, hard-deletes from CRDB (if not under legal hold)
6. Emits `archival.completed.v1` to outbox for audit

GCS bucket lifecycle: Coldline → Archive after 1 year → permanent retention
(regulated tables) or deletion (LGPD-eligible tables).

## Right-to-erasure workflow (LGPD Art. 18 IV)

```
DPO receives request → routes via secure channel
   ↓
Verify identity (CPF + 2FA via IdentityOS)
   ↓
Classification:
   ├─ If under regulatory hold (BACEN 5y / 10y) → REJECT with citation, notify ANPD
   ├─ If overlapping legal basis (KYC active) → DEFER until basis expires
   └─ If erasable → proceed
        ↓
   Execute via cmd/erasure-worker:
     - UPDATE actor SET name='[REDACTED]', tax_id='[REDACTED]', email='[REDACTED]', ...
     - DELETE from quote_streams WHERE requester_id=...
     - UPDATE counterparties SET name='[REDACTED]', beneficial_owner='[REDACTED]' WHERE id IN ...
     - Audit-log every change with audit_event(type='LGPD_ERASURE', subject_id=...)
        ↓
   Notify subject within 15 days (Art. 19 LGPD)
        ↓
   Update DPO log
```

Workflow code: `cmd/erasure-worker/` (not yet implemented; tracked as future
work in `.base/plans/00-governance/lgpd-backlog.md`).

## Data Protection Officer (DPO) responsibilities

- Single point of contact for ANPD (Autoridade Nacional de Proteção de Dados)
- Maintains the data inventory (this document)
- Quarterly review of retention vs. actual archival/deletion
- Tracks erasure requests in `.audit-bundles/lgpd-requests/YYYY/`
- Annual DPIA (Data Protection Impact Assessment) for any new field collecting PII

## Operational checklist

| Activity | Frequency | Owner | Evidence |
|----------|-----------|-------|----------|
| Archival job ran successfully | Daily | Platform on-call | Grafana panel `archive_jobs_completed_total` |
| Retention review (table-by-table) | Quarterly | DPO + Compliance | `.audit-bundles/lgpd-reviews/YYYY-Qn.md` |
| Right-to-erasure requests SLA (15 days) | Per-request | DPO | `.audit-bundles/lgpd-requests/` |
| DPIA for new PII fields | Per-field | DPO | `.audit-bundles/dpia/YYYY-MM-<field>.md` |
| Restore-from-archive drill | Annual | Platform + DPO | Drill writeup |

## Cross-references

- ISO 27001 controls: 5.34 (privacy + PII protection), 8.10 (deletion), 8.11 (masking), 8.12 (DLP)
- `docs/security/iso27001-controls-mapping.md`
- `docs/security/incident-response.md` § PII breach scenario (S-3)
- `docs/security/dr-runbook.md` § restore-from-archive
- `docs/operations/cost-allocation.md` (archive bucket cost)

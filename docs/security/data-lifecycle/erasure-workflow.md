# LGPD Right-to-Erasure Workflow

> Operational procedure for handling Art. 18 IV (LGPD) erasure requests.
> Target SLA: respond within 15 days (Art. 19). Owner: DPO.

## Intake

Requests arrive via:
- DPO email mailbox (LGPD-only; segregated)
- Customer portal "exercise data rights" button (planned)
- Postal mail (rare; redirect to DPO email)

Per request, create a tracking ticket `LGPD-YYYY-NNNN` in the DPO's secure
issue tracker.

## Stage 1 — Identity verification (within 3 days)

Required:
- CPF/CNPJ of the data subject
- Government-issued ID copy
- Second factor: SMS code via IdentityOS OR notarised request

If unable to verify within 7 days: reject with cause (Art. 18 § 1º — entity must
demonstrate identity to exercise the right).

Tracker fields:
```yaml
ticket: LGPD-2026-0001
received_at: 2026-05-24T10:30:00-03:00
subject_cpf: hashed-sha256-of-cpf
verification_method: identityos-sms
verified_at: 2026-05-26T14:00:00-03:00
```

## Stage 2 — Eligibility check (within 7 days)

Run `scripts/lgpd-eligibility.sh <subject-id>` to query CRDB across all tables:

```bash
bash scripts/lgpd-eligibility.sh <subject-id>
```

Output for each table the subject appears in:
- table name
- row count
- earliest occurred_at
- regulatory hold status (calculated from `data-lifecycle/README.md` retention)
- erasure eligibility decision (ELIGIBLE / DEFERRED_UNTIL_<date> / FROZEN_REGULATORY)

If subject has rows in `fx_trades`, `bacen_reports`, `audit_events` etc. → those
rows are FROZEN (regulatory hold). PII fields in `actors` may still be redactable
without affecting the hold (the trade record stays; the name on it is replaced
with `[REDACTED PER LGPD ART 18 IV LGPD-2026-0001]`).

## Stage 3 — Execution (within 12 days)

The executor is [`cmd/erasure-worker/`](../../../cmd/erasure-worker/main.go).
Contract for `--plan` input is the JSON Schema at
[`schemas/erasure-plan-v1.json`](../../../schemas/erasure-plan-v1.json) —
plans are JSON, not YAML, to avoid an external parser in the security path.
The DPO console may render YAML for human review and convert before signing.

Example plan (matches `erasure-plan-v1.json`):

```json
{
  "ticket": "LGPD-2026-0001",
  "subject_ref": "hashed-cpf-or-uuid",
  "approvals": ["dpo", "compliance_officer"],
  "operations": [
    {
      "table": "actors",
      "where": "actor_id IN ('uuid-1', 'uuid-2')",
      "op": "redact",
      "fields": ["display_name", "external_sub"]
    },
    {
      "table": "counterparties",
      "where": "counterparty_id IN ('uuid-3')",
      "op": "hard_delete"
    }
  ]
}
```

DPO + Compliance Officer co-sign the plan. Then:

```bash
# Optional but recommended — exact row count preview per op.
EXCHANGEOS_DB_DSN="postgres://..." \
  ./bin/erasure-worker --ticket LGPD-2026-0001 --plan plan.json --dry-run

# Execute — requires THREE independent guards:
EXCHANGEOS_ERASURE_APPROVERS="dpo,compliance_officer" \
EXCHANGEOS_ERASURE_CONFIRM="YES-I-MEAN-IT" \
EXCHANGEOS_DB_DSN="postgres://..." \
EXCHANGEOS_OPERATOR_TENANT_ID="<platform-tenant-uuid>" \
  ./bin/erasure-worker --ticket LGPD-2026-0001 --plan plan.json --execute
```

Every operation:
- Runs inside its own CRDB transaction with `SET TRANSACTION PRIORITY HIGH`
- Emits one `audit_events` row with `event_type='LGPD_ERASURE_OP'` carrying
  the full op payload (ticket, table, rows_affected, before/after hashes)
- After the last op commits, emits a final `audit_events` row
  (`LGPD_ERASURE_COMPLETED`) + an `outbox_events` row
  (`event_name='lgpd.erasure_completed.v1'`, topic
  `exchangeos.lgpd.erasure.events.v1`) atomically in one tx so downstream
  (LedgerOS, ComplOS) consumers can apply equivalent erasure.

Production deploy: [`erasure-worker-cronjob.yaml`](../../../deploy/helm/exchangeos/templates/erasure-worker-cronjob.yaml)
ships a permanently-suspended CronJob; on-call triggers it via
`kubectl create job --from=cronjob/erasure-worker`. See the runbook entry in
[`docs/operations/runbook-index.md`](../../operations/runbook-index.md) under
**Security & compliance** for the full invocation.

Each request's evidence (intake form, eligibility report, signed plan,
executor logs, completion proof) lands under
[`.audit-bundles/lgpd-requests/<ticket>/`](../../../.audit-bundles/lgpd-requests/README.md) —
7-year retention per regulatory baseline.

## Stage 4 — Response to subject (within 15 days)

Send signed confirmation:
- What was erased / redacted
- What was retained + LEGAL BASIS (cite the regulation)
- Subject's right to escalate to ANPD if dissatisfied

Save the signed confirmation under `.audit-bundles/lgpd-requests/YYYY/LGPD-YYYY-NNNN/`.

## Stage 5 — Post-completion audit

Within 30 days, DPO verifies:
- No queries against erased fields succeed (sample-query test)
- Backups older than the request date contain the data (expected — retention not
  retroactive to backups; backups age out per schedule)
- After full backup-retention period (90 days), erasure is "complete" — record
  closure in tracker

## Edge cases

### Subject appears only in audit logs

Audit logs cannot be edited (would break integrity). Respond explaining the
regulatory basis. Optionally, encrypt the row with a tenant-key + destroy the
key on next legal-basis expiry → "crypto-shredding" (planned, ISO 27001 8.10).

### Subject is also an employee actor

Employee records have separate retention (10 years per CLT). DPO consults HR
before any erasure of `actors` rows with role='EMPLOYEE'.

### Request volume spike

If > 5 simultaneous requests, DPO may invoke Art. 19 § 4º (extension by up to
2 months with notification) — must notify both the subject AND ANPD.

## Reporting

Quarterly to leadership:
- Total requests received
- Within-SLA % (target ≥ 95%)
- Deferred / rejected breakdown
- Average time-to-completion
- Escalations to ANPD (target: zero)

Annual to ANPD per regulator request (Art. 38 LGPD).

## Cross-references

- `README.md` (data classification + retention)
- `docs/security/iso27001-controls-mapping.md` controls 5.34, 8.10
- `docs/security/incident-response.md` § S-3 PII breach
- `.base/plans/00-governance/lgpd-backlog.md` for `cmd/erasure-worker` implementation tracking

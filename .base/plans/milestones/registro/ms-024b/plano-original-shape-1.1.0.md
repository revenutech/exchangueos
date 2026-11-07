# MS-024b — Archival Cron Worker

| Field | Value |
|-------|-------|
| **Code** | MS-024b |
| **Name** | archival-cron-worker |
| **Phase** | F-OPS-PROD |
| **Sprint** | 1 of MS-024 cycle |
| **Status** | BACKLOG |
| **Owner** | Platform |
| **Dependencies** | MS-024h (postgres repos), GCS bucket terraform applied |

## Why this milestone

`docs/security/data-lifecycle/archival-cron.md` specs `cmd/archiver` (02:00 SP nightly) + `archive_policy` config table + GCS Coldline writes with SHA256 manifest, but no code exists. CRDB will retain everything forever until built.

## Description

Implement `cmd/archiver/main.go` as a Kubernetes CronJob runner that ages CRDB rows out to GCS Coldline per data-driven policy. Includes parquet writer + SHA256 manifest + retention-after hard-delete + outbox event emission + quarterly restore-drill harness.

## Acceptance Criteria

- [ ] `cmd/archiver/main.go` accepting `--policy-table archive_policy` + `--batch-size 10000` + `--max-tables 25`
- [ ] Migration 000010 creating `archive_policy` + `archive_manifest` tables with seed values (25 tables per data-lifecycle/README.md)
- [ ] Parquet bulk-write per batch to `gs://exchangeos-archive-<env>/<gcs_prefix>YYYY/MM/DD/batch-<uuid>.parquet.gz` with zstd compression
- [ ] SHA256 computed + stored in `archive_manifest`
- [ ] `UPDATE <table> SET archived_at = now()` after successful upload
- [ ] Hard-delete after `retention_after` for non-legal-hold tables
- [ ] Outbox event `archival.batch_completed.v1` per batch
- [ ] Metrics: `archive_jobs_completed_total`, `archive_batch_rows`, `archive_batch_bytes`, `archive_pending_rows{table}`
- [ ] Helm `templates/archiver-cronjob.yaml` running 05:00 UTC daily (= 02:00 SP year-round)
- [ ] Restore-from-archive harness `scripts/archive-restore-test.sh` for quarterly drill
- [ ] Integration test covering: archive batch → manifest written → rows marked → hard-delete after retention
- [ ] Grafana panel added to FinOps dashboard showing storage growth + cost projection

## Deliverables

- `cmd/archiver/main.go`
- `internal/archive/policy.go` (loads + caches `archive_policy`)
- `internal/archive/writer.go` (parquet + GCS + SHA256)
- `internal/archive/repository.go` (CRDB queries + manifest writes)
- `migrations/000010_create_archive_tables.up.sql` + `.down.sql`
- `scripts/archive-restore-test.sh`
- `deploy/helm/exchangeos/templates/archiver-cronjob.yaml`
- `tests/integration/archiver_test.go`

## Cross-References

- `docs/security/data-lifecycle/archival-cron.md` — spec
- `docs/security/data-lifecycle/README.md` — retention schedule per table
- `docs/operations/cost-allocation.md` — GCS bucket labelling
- ISO 27001 controls 8.10, 8.13

// Package postgres — pgx/v5 backed repositories for the admin bounded context
// against migration 000008_create_compliance_admin.
//
// Aggregates persisted: SystemEvent (append-only), EODJob (UPSERT with
// optimistic version + UNIQUE (tenant_id, business_date)).
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/modules/admin/application"
	"github.com/revenutech/exchangeos/modules/admin/domain"
)

var (
	_ application.EventRepo   = (*EventRepo)(nil)
	_ application.EODJobRepo  = (*EODJobRepo)(nil)
)

// ─── EventRepo ─────────────────────────────────────────────────────────────

type EventRepo struct{ pool *pgxpool.Pool }

func NewEventRepo(pool *pgxpool.Pool) *EventRepo { return &EventRepo{pool: pool} }

func (r *EventRepo) Save(ctx context.Context, e *domain.SystemEvent) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO system_events (
			event_id, code, component, description, at, iso20022_ref
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		e.ID(), string(e.Code()), e.Component(), nullableString(e.Description()),
		e.At(), nullableString(e.ISO20022Ref()),
	)
	if err != nil {
		return fmt.Errorf("system_event.save: %w", err)
	}
	return nil
}

func (r *EventRepo) List(ctx context.Context, limit int) ([]*domain.SystemEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT event_id, code, component,
		       COALESCE(description, ''),
		       at,
		       COALESCE(iso20022_ref, '')
		FROM system_events
		ORDER BY at DESC
		LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("system_event.list: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.SystemEvent, 0, limit)
	for rows.Next() {
		var (
			id              uuid.UUID
			code            string
			component, desc string
			at              time.Time
			iso20022Ref     string
		)
		if err := rows.Scan(&id, &code, &component, &desc, &at, &iso20022Ref); err != nil {
			return nil, fmt.Errorf("system_event.scan: %w", err)
		}
		out = append(out, domain.ReconstituteSystemEvent(
			id, domain.EventCode(code), component, desc, at, iso20022Ref,
		))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("system_event.iter: %w", err)
	}
	return out, nil
}

// ─── EODJobRepo ────────────────────────────────────────────────────────────

type EODJobRepo struct{ pool *pgxpool.Pool }

func NewEODJobRepo(pool *pgxpool.Pool) *EODJobRepo { return &EODJobRepo{pool: pool} }

// Save upserts the job by job_id with optimistic version check. The
// (tenant_id, business_date) UNIQUE constraint guards against duplicate
// inserts for the same business day; that surface as application.ErrInvalidInput
// rather than ErrConflict because the application layer enforces the conflict
// rule via the FindByDate-then-Save pattern.
func (r *EODJobRepo) Save(ctx context.Context, j *domain.EODJob) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO eod_jobs (
			job_id, tenant_id, business_date, status,
			started_at, completed_at, failure_reason, steps_done, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (job_id) DO UPDATE SET
			status         = EXCLUDED.status,
			started_at     = EXCLUDED.started_at,
			completed_at   = EXCLUDED.completed_at,
			failure_reason = EXCLUDED.failure_reason,
			steps_done     = EXCLUDED.steps_done,
			version        = EXCLUDED.version,
			updated_at     = current_timestamp()
		WHERE eod_jobs.version = EXCLUDED.version - 1`,
		j.ID(), j.TenantID(), j.BusinessDate(), string(j.Status()),
		nullableTime(j.StartedAt()), nullableTime(j.CompletedAt()),
		nullableString(j.FailureReason()), j.StepsDone(), j.Version(),
	)
	if err != nil {
		return fmt.Errorf("eod_job.save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrInvalidInput
	}
	return nil
}

func (r *EODJobRepo) Get(ctx context.Context, id uuid.UUID) (*domain.EODJob, error) {
	return r.scanOne(ctx, `WHERE job_id = $1`, id)
}

func (r *EODJobRepo) FindByDate(ctx context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.EODJob, error) {
	day := time.Date(businessDate.Year(), businessDate.Month(), businessDate.Day(), 0, 0, 0, 0, time.UTC)
	return r.scanOne(ctx, `WHERE tenant_id = $1 AND business_date = $2`, tenantID, day)
}

func (r *EODJobRepo) scanOne(ctx context.Context, where string, args ...any) (*domain.EODJob, error) {
	q := `SELECT job_id, tenant_id, business_date, status,
	             COALESCE(started_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
	             COALESCE(completed_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
	             COALESCE(failure_reason, ''),
	             steps_done,
	             version
	      FROM eod_jobs ` + where
	var (
		id, tenantID uuid.UUID
		businessDate time.Time
		status       string
		startedAt    time.Time
		completedAt  time.Time
		failure      string
		steps        []string
		version      int
	)
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&id, &tenantID, &businessDate, &status,
		&startedAt, &completedAt, &failure, &steps, &version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("eod_job.scan: %w", err)
	}
	return domain.ReconstituteEODJob(
		id, tenantID, businessDate, domain.EODStatus(status),
		startedAt, completedAt, failure, steps, version,
	), nil
}

// ─── helpers ───────────────────────────────────────────────────────────────

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

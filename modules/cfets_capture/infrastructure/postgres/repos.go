// Package postgres — pgx/v5 backed CFETSCaptureRepo against migration
// 000012_create_cfets.cfets_captures.
//
// UPSERT on capture_id with optimistic version check guarding the WHERE clause
// so concurrent writers cannot stomp each other's transitions.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/modules/cfets_capture/application"
	"github.com/revenutech/exchangeos/modules/cfets_capture/domain"
)

var _ application.Repository = (*CFETSCaptureRepo)(nil)

type CFETSCaptureRepo struct{ pool *pgxpool.Pool }

func NewCFETSCaptureRepo(pool *pgxpool.Pool) *CFETSCaptureRepo {
	return &CFETSCaptureRepo{pool: pool}
}

// Save upserts the capture by capture_id with optimistic version check.
func (r *CFETSCaptureRepo) Save(ctx context.Context, c *domain.CFETSCapture) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO cfets_captures (
			capture_id, tenant_id, trade_id, submitter_ref, cfets_deal_id,
			status, submitted_at, ack_at, notified_at, rejection_reason, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (capture_id) DO UPDATE SET
			cfets_deal_id    = EXCLUDED.cfets_deal_id,
			status           = EXCLUDED.status,
			submitted_at     = EXCLUDED.submitted_at,
			ack_at           = EXCLUDED.ack_at,
			notified_at      = EXCLUDED.notified_at,
			rejection_reason = EXCLUDED.rejection_reason,
			version          = EXCLUDED.version,
			updated_at       = current_timestamp()
		WHERE cfets_captures.version = EXCLUDED.version - 1`,
		c.ID(), c.TenantID(), c.TradeID(), c.SubmitterRef(),
		nullableString(c.CFETSDealID()),
		string(c.Status()),
		nullableTime(c.SubmittedAt()), nullableTime(c.AckAt()), nullableTime(c.NotifiedAt()),
		nullableString(c.RejectionReason()),
		c.Version(),
	)
	if err != nil {
		return fmt.Errorf("cfets_capture.save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrInvalidInput
	}
	return nil
}

func (r *CFETSCaptureRepo) Get(ctx context.Context, id uuid.UUID) (*domain.CFETSCapture, error) {
	var (
		cid, tenantID, tradeID uuid.UUID
		submitterRef, dealID   string
		status                 string
		submittedAt            time.Time
		ackAt                  time.Time
		notifiedAt             time.Time
		rejection              string
		version                int
	)
	err := r.pool.QueryRow(ctx, `
		SELECT capture_id, tenant_id, trade_id, submitter_ref,
		       COALESCE(cfets_deal_id, ''),
		       status,
		       COALESCE(submitted_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(ack_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(notified_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(rejection_reason, ''),
		       version
		FROM cfets_captures
		WHERE capture_id = $1`,
		id,
	).Scan(&cid, &tenantID, &tradeID, &submitterRef, &dealID, &status,
		&submittedAt, &ackAt, &notifiedAt, &rejection, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cfets_capture.get: %w", err)
	}
	return domain.ReconstituteCapture(
		cid, tenantID, tradeID, submitterRef, dealID,
		domain.CaptureStatus(status),
		submittedAt, ackAt, notifiedAt, rejection, version,
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

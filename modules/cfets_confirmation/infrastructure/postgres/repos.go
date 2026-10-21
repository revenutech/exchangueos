// Package postgres — pgx/v5 backed CFETSConfirmationRepo against migration
// 000012_create_cfets.cfets_confirmations.
//
// UPSERT on confirmation_id with optimistic version check.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/modules/cfets_confirmation/application"
	"github.com/revenutech/exchangeos/modules/cfets_confirmation/domain"
)

var _ application.Repository = (*CFETSConfirmationRepo)(nil)

type CFETSConfirmationRepo struct{ pool *pgxpool.Pool }

func NewCFETSConfirmationRepo(pool *pgxpool.Pool) *CFETSConfirmationRepo {
	return &CFETSConfirmationRepo{pool: pool}
}

func (r *CFETSConfirmationRepo) Save(ctx context.Context, c *domain.CFETSConfirmation) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO cfets_confirmations (
			confirmation_id, tenant_id, trade_id, cfets_deal_id,
			status, requested_at, confirmed_at, rejection_reason, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (confirmation_id) DO UPDATE SET
			status           = EXCLUDED.status,
			confirmed_at     = EXCLUDED.confirmed_at,
			rejection_reason = EXCLUDED.rejection_reason,
			version          = EXCLUDED.version,
			updated_at       = current_timestamp()
		WHERE cfets_confirmations.version = EXCLUDED.version - 1`,
		c.ID(), c.TenantID(), c.TradeID(), c.CFETSDealID(),
		string(c.Status()), c.RequestedAt(), nullableTime(c.ConfirmedAt()),
		nullableString(c.RejectionReason()), c.Version(),
	)
	if err != nil {
		return fmt.Errorf("cfets_confirmation.save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrInvalidInput
	}
	return nil
}

func (r *CFETSConfirmationRepo) Get(ctx context.Context, id uuid.UUID) (*domain.CFETSConfirmation, error) {
	var (
		cid, tenantID, tradeID uuid.UUID
		dealID                 string
		status                 string
		requestedAt            time.Time
		confirmedAt            time.Time
		rejection              string
		version                int
	)
	err := r.pool.QueryRow(ctx, `
		SELECT confirmation_id, tenant_id, trade_id, cfets_deal_id,
		       status, requested_at,
		       COALESCE(confirmed_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(rejection_reason, ''),
		       version
		FROM cfets_confirmations
		WHERE confirmation_id = $1`,
		id,
	).Scan(&cid, &tenantID, &tradeID, &dealID, &status,
		&requestedAt, &confirmedAt, &rejection, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cfets_confirmation.get: %w", err)
	}
	return domain.ReconstituteConfirmation(
		cid, tenantID, tradeID, dealID,
		domain.ConfirmationStatus(status),
		requestedAt, confirmedAt, rejection, version,
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

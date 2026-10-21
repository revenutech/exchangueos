// Package postgres — pgx/v5 PayInRepo against migration 000006_create_settlement.
//
// Reference impl for the MS-024h pattern. The other 5 BCs (netreport / compliance /
// admin / cfets_capture / cfets_confirmation) will follow this exact shape:
//   - constructor `NewXxxRepo(pool)` returning concrete type that satisfies
//     the application.Repository interface;
//   - Save uses UPSERT (ON CONFLICT (id) DO UPDATE) with optimistic version
//     bump checked in WHERE clause for compound writes;
//   - Get returns ErrNotFound on pgx.ErrNoRows;
//   - listing methods read with ORDER BY for stable output;
//   - nullable columns coalesced via NULLIF / COALESCE on read.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/payin/application"
	"github.com/revenutech/exchangeos/modules/payin/domain"
)

// Compile-time interface satisfaction check.
var _ application.Repository = (*PayInRepo)(nil)

type PayInRepo struct{ pool *pgxpool.Pool }

func NewPayInRepo(pool *pgxpool.Pool) *PayInRepo { return &PayInRepo{pool: pool} }

// Save upserts the instruction. Optimistic concurrency: the version of the
// in-memory aggregate must equal the row version + 1 (or be 1 if the row
// does not exist). Returns ErrConflict on version mismatch.
func (r *PayInRepo) Save(ctx context.Context, p *domain.PayInInstruction) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO payin_instructions (
			instruction_id, tenant_id, cycle_id, currency, amount,
			band, deadline, status, submitted_at, confirmed_at,
			failure_reason, version
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12
		)
		ON CONFLICT (instruction_id) DO UPDATE SET
			status         = EXCLUDED.status,
			submitted_at   = EXCLUDED.submitted_at,
			confirmed_at   = EXCLUDED.confirmed_at,
			failure_reason = EXCLUDED.failure_reason,
			version        = EXCLUDED.version,
			updated_at     = current_timestamp()
		WHERE payin_instructions.version = EXCLUDED.version - 1
	`,
		p.ID(), p.TenantID(), p.CycleID(), p.Currency(), p.Amount(),
		string(p.Band()), p.Deadline(), string(p.Status()),
		nullableTime(p.SubmittedAt()), nullableTime(p.ConfirmedAt()),
		nullableString(p.FailureReason()), p.Version(),
	)
	if err != nil {
		return fmt.Errorf("payin.save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrInvalidInput // pessimistic: caller likely raced
	}
	return nil
}

// Get returns the instruction by ID, or application.ErrNotFound.
func (r *PayInRepo) Get(ctx context.Context, id uuid.UUID) (*domain.PayInInstruction, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT instruction_id, tenant_id, cycle_id, currency, amount,
		       band, deadline, status,
		       COALESCE(submitted_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(confirmed_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(failure_reason, ''),
		       version
		FROM payin_instructions
		WHERE instruction_id = $1
	`, id)
	return scanPayIn(row)
}

// ListByCycle returns instructions for one cycle ordered by (currency, deadline).
func (r *PayInRepo) ListByCycle(ctx context.Context, cycleID uuid.UUID) ([]*domain.PayInInstruction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT instruction_id, tenant_id, cycle_id, currency, amount,
		       band, deadline, status,
		       COALESCE(submitted_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(confirmed_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(failure_reason, ''),
		       version
		FROM payin_instructions
		WHERE cycle_id = $1
		ORDER BY currency ASC, deadline ASC
	`, cycleID)
	if err != nil {
		return nil, fmt.Errorf("payin.list: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.PayInInstruction, 0, 16)
	for rows.Next() {
		p, err := scanPayIn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payin.iter: %w", err)
	}
	return out, nil
}

// ─── helpers ───────────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanPayIn(s scannable) (*domain.PayInInstruction, error) {
	var (
		id, tenantID, cycleID  uuid.UUID
		currency               string
		amount                 pgtype.Numeric
		band, status, reason   string
		deadline, submitted, confirmed time.Time
		version                int
	)
	if err := s.Scan(
		&id, &tenantID, &cycleID, &currency, &amount,
		&band, &deadline, &status, &submitted, &confirmed, &reason, &version,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, fmt.Errorf("payin.scan: %w", err)
	}
	dec, err := numericToDecimal(amount)
	if err != nil {
		return nil, fmt.Errorf("payin.amount: %w", err)
	}
	return domain.ReconstitutePayIn(
		id, tenantID, cycleID, currency, dec,
		domain.DeadlineBand(band), deadline,
		domain.PayInStatus(status),
		submitted, confirmed, reason, version,
	), nil
}

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

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}
	// pgtype.Numeric → string → decimal.Decimal round-trip preserves precision.
	v, err := n.Value()
	if err != nil {
		return decimal.Zero, err
	}
	str, ok := v.(string)
	if !ok {
		return decimal.Zero, fmt.Errorf("expected string, got %T", v)
	}
	return decimal.NewFromString(str)
}

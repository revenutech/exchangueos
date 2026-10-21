// Package postgres — pgx/v5 CycleRepo against migration 000006.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/modules/cls_settlement/application"
	"github.com/revenutech/exchangeos/modules/cls_settlement/domain"
)

type CycleRepo struct{ pool *pgxpool.Pool }

func NewCycleRepo(pool *pgxpool.Pool) *CycleRepo { return &CycleRepo{pool: pool} }

func (r *CycleRepo) Save(ctx context.Context, c *domain.CLSCycle) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cycles.tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO cls_cycles (
			cycle_id, tenant_id, cycle_date, status,
			opened_at, pin1_deadline, pin2_deadline, pin3_deadline,
			scheduled_close, closed_at, failure_reason, version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (cycle_id) DO UPDATE SET
			status         = EXCLUDED.status,
			closed_at      = EXCLUDED.closed_at,
			failure_reason = EXCLUDED.failure_reason,
			version        = EXCLUDED.version,
			updated_at     = current_timestamp()`,
		c.ID(), c.TenantID(), c.CycleDate(), string(c.Status()),
		c.OpenedAt(), deadlineOrZero(c, "PIN1"), deadlineOrZero(c, "PIN2"), deadlineOrZero(c, "PIN3"),
		c.ScheduledClose(), nullableTime(c.ClosedAt()), nullableString(c.FailureReason()), c.Version(),
	)
	if err != nil {
		return fmt.Errorf("cycles.upsert: %w", err)
	}

	// Replace the trade_ids set transactionally (delete-then-insert keeps the FK
	// list aligned with the aggregate; per-trade transitions remain idempotent).
	if _, err := tx.Exec(ctx, `DELETE FROM cls_cycle_trades WHERE cycle_id = $1`, c.ID()); err != nil {
		return fmt.Errorf("cycle_trades.clear: %w", err)
	}
	for _, tid := range c.TradeIDs() {
		if _, err := tx.Exec(ctx,
			`INSERT INTO cls_cycle_trades (cycle_id, trade_id) VALUES ($1, $2)
			 ON CONFLICT (cycle_id, trade_id) DO NOTHING`, c.ID(), tid); err != nil {
			return fmt.Errorf("cycle_trades.insert: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *CycleRepo) Get(ctx context.Context, id uuid.UUID) (*domain.CLSCycle, error) {
	return r.scanOne(ctx, `WHERE cycle_id = $1`, id)
}

func (r *CycleRepo) FindByDate(ctx context.Context, tenantID uuid.UUID, businessDate time.Time) (*domain.CLSCycle, error) {
	day := time.Date(businessDate.Year(), businessDate.Month(), businessDate.Day(), 0, 0, 0, 0, time.UTC)
	return r.scanOne(ctx, `WHERE tenant_id = $1 AND cycle_date = $2`, tenantID, day)
}

func (r *CycleRepo) scanOne(ctx context.Context, where string, args ...any) (*domain.CLSCycle, error) {
	var (
		id, tenantID       uuid.UUID
		cycleDate          time.Time
		status             string
		openedAt           time.Time
		pin1, pin2, pin3   time.Time
		scheduledClose     time.Time
		closedAtNullable   *time.Time
		failureReason      string
		version            int
	)
	q := `SELECT cycle_id, tenant_id, cycle_date, status, opened_at,
	             pin1_deadline, pin2_deadline, pin3_deadline, scheduled_close,
	             closed_at, COALESCE(failure_reason, ''), version
	      FROM cls_cycles ` + where
	row := r.pool.QueryRow(ctx, q, args...)
	if err := row.Scan(&id, &tenantID, &cycleDate, &status, &openedAt,
		&pin1, &pin2, &pin3, &scheduledClose, &closedAtNullable, &failureReason, &version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, fmt.Errorf("cycles.scan: %w", err)
	}

	// Hydrate attached trade ids.
	tradeRows, err := r.pool.Query(ctx,
		`SELECT trade_id FROM cls_cycle_trades WHERE cycle_id = $1 ORDER BY trade_id`, id)
	if err != nil {
		return nil, fmt.Errorf("cycle_trades.query: %w", err)
	}
	defer tradeRows.Close()
	var tradeIDs []uuid.UUID
	for tradeRows.Next() {
		var tid uuid.UUID
		if err := tradeRows.Scan(&tid); err != nil {
			return nil, fmt.Errorf("cycle_trades.scan: %w", err)
		}
		tradeIDs = append(tradeIDs, tid)
	}

	var closedAt time.Time
	if closedAtNullable != nil {
		closedAt = *closedAtNullable
	}
	return domain.ReconstituteCycle(
		id, tenantID, cycleDate, domain.CycleStatus(status),
		openedAt, pin1, pin2, pin3, scheduledClose, closedAt,
		failureReason, tradeIDs, version,
	), nil
}

// Helpers
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func deadlineOrZero(c *domain.CLSCycle, band string) time.Time {
	t, err := c.DeadlineFor(band)
	if err != nil {
		return time.Time{}
	}
	return t
}

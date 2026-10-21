// Package postgres — pgx/v5 PositionRepo against migration 000007.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/position/application"
	"github.com/revenutech/exchangeos/modules/position/domain"
)

type PositionRepo struct{ pool *pgxpool.Pool }

func NewPositionRepo(pool *pgxpool.Pool) *PositionRepo { return &PositionRepo{pool: pool} }

func (r *PositionRepo) Save(ctx context.Context, p *domain.Position) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO positions (
			position_id, tenant_id, currency, long_amount, short_amount, net_amount, as_of, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, currency) DO UPDATE SET
			long_amount  = EXCLUDED.long_amount,
			short_amount = EXCLUDED.short_amount,
			net_amount   = EXCLUDED.net_amount,
			as_of        = EXCLUDED.as_of,
			version      = EXCLUDED.version,
			updated_at   = current_timestamp()`,
		p.ID(), p.TenantID(), p.Currency(), p.Long(), p.Short(), p.Net(), p.AsOf(), p.Version(),
	)
	if err != nil {
		return fmt.Errorf("positions.save: %w", err)
	}
	return nil
}

func (r *PositionRepo) Get(ctx context.Context, tenantID uuid.UUID, currency string) (*domain.Position, error) {
	var (
		id              uuid.UUID
		long, short     decimal.Decimal
		asOf            time.Time
		version         int
	)
	err := r.pool.QueryRow(ctx, `
		SELECT position_id, long_amount, short_amount, as_of, version
		FROM positions WHERE tenant_id = $1 AND currency = $2`,
		tenantID, strings.ToUpper(strings.TrimSpace(currency)),
	).Scan(&id, &long, &short, &asOf, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("positions.get: %w", err)
	}
	return domain.ReconstitutePosition(id, tenantID, strings.ToUpper(currency), long, short, asOf, version), nil
}

func (r *PositionRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Position, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT position_id, currency, long_amount, short_amount, as_of, version
		FROM positions WHERE tenant_id = $1 ORDER BY currency`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("positions.list: %w", err)
	}
	defer rows.Close()
	var out []*domain.Position
	for rows.Next() {
		var (
			id          uuid.UUID
			currency    string
			long, short decimal.Decimal
			asOf        time.Time
			version     int
		)
		if err := rows.Scan(&id, &currency, &long, &short, &asOf, &version); err != nil {
			return nil, fmt.Errorf("positions.scan: %w", err)
		}
		out = append(out, domain.ReconstitutePosition(id, tenantID, currency, long, short, asOf, version))
	}
	return out, rows.Err()
}

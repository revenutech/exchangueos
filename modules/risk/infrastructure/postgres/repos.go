// Package postgres — pgx/v5 LimitRepo against migration 000007.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/risk/application"
	"github.com/revenutech/exchangeos/modules/risk/domain"
)

type LimitRepo struct{ pool *pgxpool.Pool }

func NewLimitRepo(pool *pgxpool.Pool) *LimitRepo { return &LimitRepo{pool: pool} }

func (r *LimitRepo) Save(ctx context.Context, l *domain.Limit) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO risk_limits (limit_id, tenant_id, limit_type, scope, cap, utilised, currency, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (limit_id) DO UPDATE SET
			utilised   = EXCLUDED.utilised,
			version    = EXCLUDED.version,
			updated_at = current_timestamp()`,
		l.ID(), l.TenantID(), string(l.Type()), l.Scope(),
		l.Cap(), l.Utilised(), l.Currency(), l.Version(),
	)
	if err != nil {
		return fmt.Errorf("risk_limits.save: %w", err)
	}
	return nil
}

func (r *LimitRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Limit, error) {
	return r.scanOne(ctx, `WHERE limit_id = $1`, id)
}

func (r *LimitRepo) Find(ctx context.Context, tenantID uuid.UUID, t domain.LimitType, scope string) (*domain.Limit, error) {
	return r.scanOne(ctx, `WHERE tenant_id = $1 AND limit_type = $2 AND scope = $3`,
		tenantID, string(t), strings.ToUpper(scope))
}

func (r *LimitRepo) scanOne(ctx context.Context, where string, args ...any) (*domain.Limit, error) {
	var (
		id, tenantID    uuid.UUID
		limitType       string
		scope, currency string
		cap, utilised   decimal.Decimal
		version         int
	)
	q := `SELECT limit_id, tenant_id, limit_type, scope, cap, utilised, currency, version
	      FROM risk_limits ` + where
	if err := r.pool.QueryRow(ctx, q, args...).
		Scan(&id, &tenantID, &limitType, &scope, &cap, &utilised, &currency, &version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, fmt.Errorf("risk_limits.scan: %w", err)
	}
	return domain.ReconstituteLimit(id, tenantID, domain.LimitType(limitType),
		scope, currency, cap, utilised, version), nil
}

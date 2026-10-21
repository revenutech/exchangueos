// Package postgres — pgx/v5 NetReportRepo against migration 000006_create_settlement.
//
// Second of 6 BCs delivered for MS-024h. Pattern follows payin/postgres/repos.go:
// UPSERT on (cycle_id, currency) UNIQUE constraint, ListByCycle ordered by
// currency for stable output, NULL handling only where the schema permits.
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

	"github.com/revenutech/exchangeos/modules/netreport/application"
	"github.com/revenutech/exchangeos/modules/netreport/domain"
)

var _ application.Repository = (*NetReportRepo)(nil)

type NetReportRepo struct{ pool *pgxpool.Pool }

func NewNetReportRepo(pool *pgxpool.Pool) *NetReportRepo { return &NetReportRepo{pool: pool} }

// Save upserts on the (cycle_id, currency) UNIQUE constraint. NetReports are
// idempotent: regenerating overwrites with the latest gross/net values.
func (r *NetReportRepo) Save(ctx context.Context, n *domain.NetReport) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO net_reports (
			report_id, tenant_id, cycle_id, currency,
			gross_pay_in, gross_pay_out, net_settlement, trade_count, generated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (cycle_id, currency) DO UPDATE SET
			gross_pay_in   = EXCLUDED.gross_pay_in,
			gross_pay_out  = EXCLUDED.gross_pay_out,
			net_settlement = EXCLUDED.net_settlement,
			trade_count    = EXCLUDED.trade_count,
			generated_at   = EXCLUDED.generated_at
	`,
		n.ID(), n.TenantID(), n.CycleID(), n.Currency(),
		n.GrossPayIn(), n.GrossPayOut(), n.NetSettlement(),
		n.TradeCount(), n.GeneratedAt(),
	)
	if err != nil {
		return fmt.Errorf("netreport.save: %w", err)
	}
	return nil
}

// GetByCycleCcy fetches the report by (cycle_id, currency).
func (r *NetReportRepo) GetByCycleCcy(ctx context.Context, cycleID uuid.UUID, currency string) (*domain.NetReport, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT report_id, tenant_id, cycle_id, currency,
		       gross_pay_in, gross_pay_out, net_settlement, trade_count, generated_at
		FROM net_reports
		WHERE cycle_id = $1 AND currency = $2
	`, cycleID, currency)
	return scanNetReport(row)
}

// ListByCycle returns all per-currency reports for a cycle, ordered alphabetically.
func (r *NetReportRepo) ListByCycle(ctx context.Context, cycleID uuid.UUID) ([]*domain.NetReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT report_id, tenant_id, cycle_id, currency,
		       gross_pay_in, gross_pay_out, net_settlement, trade_count, generated_at
		FROM net_reports
		WHERE cycle_id = $1
		ORDER BY currency ASC
	`, cycleID)
	if err != nil {
		return nil, fmt.Errorf("netreport.list: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.NetReport, 0, 16)
	for rows.Next() {
		n, err := scanNetReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("netreport.iter: %w", err)
	}
	return out, nil
}

// ─── helpers ───────────────────────────────────────────────────────────────

type netReportScannable interface {
	Scan(dest ...any) error
}

func scanNetReport(s netReportScannable) (*domain.NetReport, error) {
	var (
		id, tenantID, cycleID uuid.UUID
		currency              string
		payIn, payOut, net    pgtype.Numeric
		tradeCount            int
		generatedAt           time.Time
	)
	if err := s.Scan(
		&id, &tenantID, &cycleID, &currency,
		&payIn, &payOut, &net, &tradeCount, &generatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, fmt.Errorf("netreport.scan: %w", err)
	}
	pi, err := numericToDecimalNR(payIn)
	if err != nil {
		return nil, fmt.Errorf("netreport.payin: %w", err)
	}
	po, err := numericToDecimalNR(payOut)
	if err != nil {
		return nil, fmt.Errorf("netreport.payout: %w", err)
	}
	ns, err := numericToDecimalNR(net)
	if err != nil {
		return nil, fmt.Errorf("netreport.net: %w", err)
	}
	return domain.ReconstituteNetReport(
		id, tenantID, cycleID, currency,
		pi, po, ns, tradeCount, generatedAt,
	), nil
}

func numericToDecimalNR(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}
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

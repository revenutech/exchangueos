// Package postgres — pgx/v5 backed TradeRepo against migration 000002.
//
// Maps fx_trades.{buyer,seller}_counterparty_id (UUID FK) → buyer_bic / seller_bic.
// For this skeleton the bic ↔ counterparty_id resolution is handled by upsert
// helpers; production wiring should denormalise via a counterparty service.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/trade/application"
	"github.com/revenutech/exchangeos/modules/trade/domain"
)

type TradeRepo struct{ pool *pgxpool.Pool }

func NewTradeRepo(pool *pgxpool.Pool) *TradeRepo { return &TradeRepo{pool: pool} }

// Save upserts an FXTrade. Buyer/seller counterparties are resolved via BIC →
// counterparties table; if not found, the row is rejected (counterparties must
// be seeded — see seeds/04_counterparties.sql).
func (r *TradeRepo) Save(ctx context.Context, t *domain.FXTrade) error {
	buyerID, err := r.cpIDByBIC(ctx, t.TenantID(), t.BuyerBIC())
	if err != nil {
		return fmt.Errorf("buyer cp lookup: %w", err)
	}
	sellerID, err := r.cpIDByBIC(ctx, t.TenantID(), t.SellerBIC())
	if err != nil {
		return fmt.Errorf("seller cp lookup: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO fx_trades (
			trade_id, tenant_id, external_ref, trade_type, status, settlement_venue,
			buyer_counterparty_id, seller_counterparty_id,
			bought_currency, bought_amount, sold_currency, sold_amount,
			deal_rate, trade_date, value_date
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15
		)
		ON CONFLICT (trade_id) DO UPDATE SET
			status     = EXCLUDED.status,
			updated_at = current_timestamp()`,
		t.ID(), t.TenantID(), nullableString(t.ExternalRef()),
		string(t.Type()), string(t.Status()), string(t.Venue()),
		buyerID, sellerID,
		t.BoughtCurrency(), t.BoughtAmount(),
		t.SoldCurrency(), t.SoldAmount(),
		t.DealRate(), t.TradeDate(), t.ValueDate(),
	)
	if err != nil {
		return fmt.Errorf("fx_trades.save: %w", err)
	}
	return nil
}

func (r *TradeRepo) Get(ctx context.Context, id uuid.UUID) (*domain.FXTrade, error) {
	var (
		tenantID                    uuid.UUID
		externalRef                 string
		tradeType, status, venue    string
		buyerBIC, sellerBIC         string
		boughtCcy, soldCcy          string
		boughtAmt, soldAmt, rate    decimal.Decimal
		tradeDate, valueDate        time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT t.tenant_id,
		       COALESCE(t.external_ref, ''),
		       t.trade_type, t.status, t.settlement_venue,
		       buyer.bic, seller.bic,
		       t.bought_currency, t.bought_amount,
		       t.sold_currency, t.sold_amount,
		       t.deal_rate, t.trade_date, t.value_date
		FROM fx_trades t
		JOIN counterparties buyer  ON buyer.counterparty_id  = t.buyer_counterparty_id
		JOIN counterparties seller ON seller.counterparty_id = t.seller_counterparty_id
		WHERE t.trade_id = $1`, id,
	).Scan(
		&tenantID, &externalRef, &tradeType, &status, &venue,
		&buyerBIC, &sellerBIC,
		&boughtCcy, &boughtAmt, &soldCcy, &soldAmt,
		&rate, &tradeDate, &valueDate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fx_trades.get: %w", err)
	}
	return domain.ReconstituteFXTrade(
		id, tenantID, externalRef,
		domain.TradeType(tradeType), domain.TradeStatus(status), domain.SettlementVenue(venue),
		buyerBIC, sellerBIC,
		boughtCcy, boughtAmt, soldCcy, soldAmt, rate,
		tradeDate, valueDate, 1,
	), nil
}

func (r *TradeRepo) List(ctx context.Context, tenantID uuid.UUID, status domain.TradeStatus, from, to time.Time, limit int) ([]*domain.FXTrade, error) {
	// Build query dynamically to keep filters optional. Use $1=tenant, $2=limit fixed,
	// then conditionally append status/date filters.
	q := `
		SELECT t.trade_id, t.tenant_id,
		       COALESCE(t.external_ref,''),
		       t.trade_type, t.status, t.settlement_venue,
		       buyer.bic, seller.bic,
		       t.bought_currency, t.bought_amount,
		       t.sold_currency, t.sold_amount,
		       t.deal_rate, t.trade_date, t.value_date
		FROM fx_trades t
		JOIN counterparties buyer  ON buyer.counterparty_id  = t.buyer_counterparty_id
		JOIN counterparties seller ON seller.counterparty_id = t.seller_counterparty_id
		WHERE t.tenant_id = $1`
	args := []any{tenantID}
	if status != "" {
		q += fmt.Sprintf(" AND t.status = $%d", len(args)+1)
		args = append(args, string(status))
	}
	if !from.IsZero() {
		q += fmt.Sprintf(" AND t.trade_date >= $%d", len(args)+1)
		args = append(args, from)
	}
	if !to.IsZero() {
		q += fmt.Sprintf(" AND t.trade_date <= $%d", len(args)+1)
		args = append(args, to)
	}
	q += fmt.Sprintf(" ORDER BY t.trade_date DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("fx_trades.list: %w", err)
	}
	defer rows.Close()

	var out []*domain.FXTrade
	for rows.Next() {
		var (
			id, tenant                  uuid.UUID
			externalRef                 string
			tradeType, st, venue        string
			buyerBIC, sellerBIC         string
			boughtCcy, soldCcy          string
			boughtAmt, soldAmt, rate    decimal.Decimal
			tradeDate, valueDate        time.Time
		)
		if err := rows.Scan(
			&id, &tenant, &externalRef,
			&tradeType, &st, &venue,
			&buyerBIC, &sellerBIC,
			&boughtCcy, &boughtAmt, &soldCcy, &soldAmt,
			&rate, &tradeDate, &valueDate,
		); err != nil {
			return nil, fmt.Errorf("fx_trades.scan: %w", err)
		}
		out = append(out, domain.ReconstituteFXTrade(
			id, tenant, externalRef,
			domain.TradeType(tradeType), domain.TradeStatus(st), domain.SettlementVenue(venue),
			buyerBIC, sellerBIC,
			boughtCcy, boughtAmt, soldCcy, soldAmt, rate,
			tradeDate, valueDate, 1,
		))
	}
	return out, rows.Err()
}

// cpIDByBIC resolves a counterparty UUID by (tenant, BIC). Returns ErrNotFound
// if no row matches — caller must seed counterparties first.
func (r *TradeRepo) cpIDByBIC(ctx context.Context, tenantID uuid.UUID, bic string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT counterparty_id FROM counterparties WHERE tenant_id = $1 AND bic = $2`,
		tenantID, bic,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("counterparty bic=%s tenant=%s: %w", bic, tenantID, application.ErrNotFound)
	}
	return id, err
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

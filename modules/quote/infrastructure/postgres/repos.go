// Package postgres — pgx/v5 backed repositories for the quote bounded context.
//
// Schema source-of-truth: migrations/000003_create_quotes.up.sql.
// UPSERT semantics on Save (DDD optimistic-concurrency check uses the version field
// in the WHERE clause for true OCC; this skeleton uses unconditional UPSERT to keep
// the surface small — tighten when concurrent writers materialise).
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

	"github.com/revenutech/exchangeos/modules/quote/application"
	"github.com/revenutech/exchangeos/modules/quote/domain"
)

// ─── QuoteRepo ─────────────────────────────────────────────────────────────

type QuoteRepo struct{ pool *pgxpool.Pool }

func NewQuoteRepo(pool *pgxpool.Pool) *QuoteRepo { return &QuoteRepo{pool: pool} }

func (r *QuoteRepo) Save(ctx context.Context, q *domain.Quote) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO quotes (
			quote_id, tenant_id, rfq_id, requester_bic, provider_bic, side,
			base_ccy, quote_ccy, notional, notional_ccy, bid, ask,
			valid_from, valid_to, venue, version
		) VALUES ($1,$2,NULL,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (quote_id) DO UPDATE SET
			notional   = EXCLUDED.notional,
			bid        = EXCLUDED.bid,
			ask        = EXCLUDED.ask,
			valid_from = EXCLUDED.valid_from,
			valid_to   = EXCLUDED.valid_to,
			venue      = EXCLUDED.venue,
			version    = EXCLUDED.version`,
		q.ID(), q.TenantID(),
		q.RequesterBIC(), q.ProviderBIC(), string(q.Side()),
		q.BaseCCY(), q.QuoteCCY(),
		q.Notional(), q.NotionalCCY(), q.Bid(), q.Ask(),
		// venue was persisted as "" here: the aggregate had no Venue() accessor,
		// so every stored quote came back with an empty venue.
		q.ValidFrom(), q.ValidTo(), q.Venue(), q.Version(),
	)
	if err != nil {
		return fmt.Errorf("quotes.save: %w", err)
	}
	return nil
}

func (r *QuoteRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Quote, error) {
	var (
		tenantID                       uuid.UUID
		requesterBIC, providerBIC      string
		side                           string
		baseCCY, quoteCCY, notionalCCY string
		notional, bid, ask             decimal.Decimal
		validFrom, validTo             time.Time
		venue                          string
		version                        int
	)
	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, requester_bic, provider_bic, side,
		       base_ccy, quote_ccy, notional, notional_ccy,
		       bid, ask, valid_from, valid_to, COALESCE(venue,''), version
		FROM quotes WHERE quote_id = $1`, id,
	).Scan(&tenantID, &requesterBIC, &providerBIC, &side,
		&baseCCY, &quoteCCY, &notional, &notionalCCY,
		&bid, &ask, &validFrom, &validTo, &venue, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("quotes.get: %w", err)
	}
	return domain.ReconstituteQuote(
		id, tenantID, requesterBIC, providerBIC, domain.Side(side),
		baseCCY, quoteCCY, notional, notionalCCY,
		bid, ask, validFrom, validTo, venue, version,
	), nil
}

// ─── RFQRepo ───────────────────────────────────────────────────────────────

type RFQRepo struct{ pool *pgxpool.Pool }

func NewRFQRepo(pool *pgxpool.Pool) *RFQRepo { return &RFQRepo{pool: pool} }

func (r *RFQRepo) Save(ctx context.Context, x *domain.RFQ) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO rfqs (rfq_id, tenant_id, requester, base_ccy, quote_ccy, status, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (rfq_id) DO UPDATE SET
			status     = EXCLUDED.status,
			version    = EXCLUDED.version,
			updated_at = current_timestamp()`,
		x.ID(), x.TenantID(),
		x.Requester(), x.BaseCCY(), x.QuoteCCY(),
		string(x.Status()), x.Version(),
	)
	if err != nil {
		return fmt.Errorf("rfqs.save: %w", err)
	}
	// The attached quote_ids are stored on the `quotes.rfq_id` FK by the
	// application layer when AttachQuoteToRFQ persists the Quote separately.
	return nil
}

func (r *RFQRepo) Get(ctx context.Context, id uuid.UUID) (*domain.RFQ, error) {
	var (
		tenantID                     uuid.UUID
		requester, baseCCY, quoteCCY string
		status                       string
		version                      int
		createdAt                    time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, requester, base_ccy, quote_ccy, status, version, created_at
		FROM rfqs WHERE rfq_id = $1`, id,
	).Scan(&tenantID, &requester, &baseCCY, &quoteCCY, &status, &version, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rfqs.get: %w", err)
	}

	// Hydrate attached quote ids.
	rows, err := r.pool.Query(ctx, `SELECT quote_id FROM quotes WHERE rfq_id = $1 ORDER BY created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("rfqs.quotes.query: %w", err)
	}
	defer rows.Close()
	var qids []uuid.UUID
	for rows.Next() {
		var qid uuid.UUID
		if err := rows.Scan(&qid); err != nil {
			return nil, fmt.Errorf("rfqs.quotes.scan: %w", err)
		}
		qids = append(qids, qid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rfqs.quotes.iter: %w", err)
	}

	return domain.ReconstituteRFQ(id, tenantID, requester, baseCCY, quoteCCY,
		domain.RFQStatus(status), qids, createdAt, version), nil
}

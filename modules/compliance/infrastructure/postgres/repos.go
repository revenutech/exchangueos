// Package postgres — pgx/v5 backed repositories for the compliance bounded
// context against migration 000008_create_compliance_admin.
//
// Aggregates persisted: Classification, IOFComputation, BACENReport, Screening.
// All queries use parameterised statements ($1, $2 …) — NEVER string interpolation.
//
// Semantics worth noting:
//   - Classification + IOF are keyed by (tenant_id, trade_id) at the
//     application layer (last-write-wins per trade). Save replaces any prior
//     row via delete-then-insert inside a single tx to keep behaviour
//     consistent with the in-memory reference impl without requiring a schema
//     migration to add a UNIQUE constraint.
//   - BACENReport is keyed by report_id with optimistic version on UPSERT.
//   - Screening is append-only.
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

	"github.com/revenutech/exchangeos/modules/compliance/application"
	"github.com/revenutech/exchangeos/modules/compliance/domain"
)

// Compile-time interface satisfaction checks.
var (
	_ application.ClassificationRepo = (*ClassificationRepo)(nil)
	_ application.IOFRepo            = (*IOFRepo)(nil)
	_ application.ReportRepo         = (*ReportRepo)(nil)
	_ application.ScreeningRepo      = (*ScreeningRepo)(nil)
)

// ─── ClassificationRepo ────────────────────────────────────────────────────

type ClassificationRepo struct{ pool *pgxpool.Pool }

func NewClassificationRepo(pool *pgxpool.Pool) *ClassificationRepo {
	return &ClassificationRepo{pool: pool}
}

func (r *ClassificationRepo) Save(ctx context.Context, c *domain.Classification) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("classification.tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM classifications WHERE tenant_id = $1 AND trade_id = $2`,
		c.TenantID(), c.TradeID(),
	); err != nil {
		return fmt.Errorf("classification.delete: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO classifications (
			classification_id, tenant_id, trade_id, code, description, nature
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID(), c.TenantID(), c.TradeID(), c.Code(), c.Description(), string(c.Nature()),
	); err != nil {
		return fmt.Errorf("classification.insert: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *ClassificationRepo) GetByTrade(ctx context.Context, tradeID uuid.UUID) (*domain.Classification, error) {
	var (
		id, tenantID, tid       uuid.UUID
		code, description, nat  string
	)
	err := r.pool.QueryRow(ctx, `
		SELECT classification_id, tenant_id, trade_id, code, description, nature
		FROM classifications
		WHERE trade_id = $1
		ORDER BY created_at DESC
		LIMIT 1`,
		tradeID,
	).Scan(&id, &tenantID, &tid, &code, &description, &nat)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("classification.get: %w", err)
	}
	return domain.ReconstituteClassification(id, tenantID, tid, code, description, domain.Nature(nat)), nil
}

// ─── IOFRepo ───────────────────────────────────────────────────────────────

type IOFRepo struct{ pool *pgxpool.Pool }

func NewIOFRepo(pool *pgxpool.Pool) *IOFRepo { return &IOFRepo{pool: pool} }

func (r *IOFRepo) Save(ctx context.Context, i *domain.IOFComputation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iof.tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM iof_computations WHERE tenant_id = $1 AND trade_id = $2`,
		i.TenantID(), i.TradeID(),
	); err != nil {
		return fmt.Errorf("iof.delete: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO iof_computations (
			iof_id, tenant_id, trade_id, operation_type,
			notional, notional_ccy, rate, iof_amount, computed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		i.ID(), i.TenantID(), i.TradeID(), i.OperationType(),
		i.Notional().String(), i.NotionalCCY(),
		i.Rate().String(), i.IOFAmount().String(), i.ComputedAt(),
	); err != nil {
		return fmt.Errorf("iof.insert: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *IOFRepo) GetByTrade(ctx context.Context, tradeID uuid.UUID) (*domain.IOFComputation, error) {
	var (
		id, tenantID, tid       uuid.UUID
		operationType, ccy      string
		notional, rate, amount  pgtype.Numeric
		computedAt              time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT iof_id, tenant_id, trade_id, operation_type,
		       notional, notional_ccy, rate, iof_amount, computed_at
		FROM iof_computations
		WHERE trade_id = $1
		ORDER BY computed_at DESC
		LIMIT 1`,
		tradeID,
	).Scan(&id, &tenantID, &tid, &operationType, &notional, &ccy, &rate, &amount, &computedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("iof.get: %w", err)
	}
	notionalDec, err := numericToDecimal(notional)
	if err != nil {
		return nil, fmt.Errorf("iof.notional: %w", err)
	}
	rateDec, err := numericToDecimal(rate)
	if err != nil {
		return nil, fmt.Errorf("iof.rate: %w", err)
	}
	amountDec, err := numericToDecimal(amount)
	if err != nil {
		return nil, fmt.Errorf("iof.amount: %w", err)
	}
	return domain.ReconstituteIOFComputation(
		id, tenantID, tid, operationType,
		notionalDec, ccy, rateDec, amountDec, computedAt,
	), nil
}

// ─── ReportRepo ────────────────────────────────────────────────────────────

type ReportRepo struct{ pool *pgxpool.Pool }

func NewReportRepo(pool *pgxpool.Pool) *ReportRepo { return &ReportRepo{pool: pool} }

// Save upserts the BACEN report by report_id with optimistic version check.
func (r *ReportRepo) Save(ctx context.Context, x *domain.BACENReport) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO bacen_reports (
			report_id, tenant_id, report_type, reference_date,
			payload_hash, status, submitted_at, responded_at,
			rejection_reason, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		ON CONFLICT (report_id) DO UPDATE SET
			status           = EXCLUDED.status,
			submitted_at     = EXCLUDED.submitted_at,
			responded_at     = EXCLUDED.responded_at,
			rejection_reason = EXCLUDED.rejection_reason,
			version          = EXCLUDED.version,
			updated_at       = current_timestamp()
		WHERE bacen_reports.version = EXCLUDED.version - 1
		   OR bacen_reports.version IS NULL`,
		x.ID(), x.TenantID(), string(x.Type()), x.ReferenceDate(),
		x.PayloadHash(), string(x.Status()),
		nullableTime(x.SubmittedAt()), nullableTime(x.RespondedAt()),
		nullableString(x.RejectionReason()), x.Version(),
	)
	if err != nil {
		return fmt.Errorf("bacen_report.save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrInvalidInput // version mismatch: caller raced
	}
	return nil
}

func (r *ReportRepo) Get(ctx context.Context, id uuid.UUID) (*domain.BACENReport, error) {
	var (
		rid, tenantID  uuid.UUID
		reportType     string
		referenceDate  time.Time
		payloadHash    string
		status         string
		submitted      time.Time
		responded      time.Time
		rejection      string
		version        int
	)
	err := r.pool.QueryRow(ctx, `
		SELECT report_id, tenant_id, report_type, reference_date,
		       payload_hash, status,
		       COALESCE(submitted_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(responded_at, TIMESTAMPTZ '0001-01-01 00:00:00+00'),
		       COALESCE(rejection_reason, ''),
		       version
		FROM bacen_reports
		WHERE report_id = $1`,
		id,
	).Scan(&rid, &tenantID, &reportType, &referenceDate, &payloadHash, &status,
		&submitted, &responded, &rejection, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bacen_report.get: %w", err)
	}
	return domain.ReconstituteBACENReport(
		rid, tenantID, domain.ReportType(reportType), referenceDate,
		payloadHash, domain.ReportStatus(status),
		submitted, responded, rejection, version,
	), nil
}

// ─── ScreeningRepo ─────────────────────────────────────────────────────────

type ScreeningRepo struct{ pool *pgxpool.Pool }

func NewScreeningRepo(pool *pgxpool.Pool) *ScreeningRepo { return &ScreeningRepo{pool: pool} }

// Save inserts a screening result. Append-only — no UPSERT.
func (r *ScreeningRepo) Save(ctx context.Context, s *domain.ScreeningResult) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO screening_results (
			screening_id, tenant_id, counterparty_bic, lei,
			risk_level, hits, screened_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID(), s.TenantID(), s.CounterpartyBIC(), nullableString(s.LEI()),
		string(s.RiskLevel()), s.Hits(), s.ScreenedAt(),
	)
	if err != nil {
		return fmt.Errorf("screening.save: %w", err)
	}
	return nil
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

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
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

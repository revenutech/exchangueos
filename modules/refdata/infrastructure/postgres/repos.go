// Package postgres — pgx/v5 backed repositories for the refdata bounded context.
//
// Schema source-of-truth: migrations/000005_create_refdata.up.sql plus
// migrations/000004_create_currency_pairs_netting.up.sql for currency_pairs.
//
// All queries use parameterised statements ($1, $2 …) — NEVER string interpolation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/modules/refdata/application"
	"github.com/revenutech/exchangeos/modules/refdata/domain"
)

// ─── CurrencyRepo ──────────────────────────────────────────────────────────

type CurrencyRepo struct{ pool *pgxpool.Pool }

func NewCurrencyRepo(pool *pgxpool.Pool) *CurrencyRepo { return &CurrencyRepo{pool: pool} }

func (r *CurrencyRepo) List(ctx context.Context, activeOnly bool) ([]*domain.Currency, error) {
	q := `SELECT code, name, minor_units, cls_eligible, cfets_eligible, active
	      FROM currencies`
	if activeOnly {
		q += ` WHERE active = true`
	}
	q += ` ORDER BY code`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("currencies.list: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Currency, 0, 32)
	for rows.Next() {
		var (
			code, name              string
			minor                   int
			cls, cfets, active      bool
		)
		if err := rows.Scan(&code, &name, &minor, &cls, &cfets, &active); err != nil {
			return nil, fmt.Errorf("currencies.scan: %w", err)
		}
		out = append(out, domain.ReconstituteCurrency(code, name, minor, cls, cfets, active))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("currencies.iter: %w", err)
	}
	return out, nil
}

func (r *CurrencyRepo) Get(ctx context.Context, code string) (*domain.Currency, error) {
	var (
		name              string
		minor             int
		cls, cfets, active bool
	)
	err := r.pool.QueryRow(ctx,
		`SELECT name, minor_units, cls_eligible, cfets_eligible, active
		 FROM currencies WHERE code = $1`, code,
	).Scan(&name, &minor, &cls, &cfets, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("currencies.get: %w", err)
	}
	return domain.ReconstituteCurrency(code, name, minor, cls, cfets, active), nil
}

// ─── CalendarRepo ──────────────────────────────────────────────────────────

type CalendarRepo struct{ pool *pgxpool.Pool }

func NewCalendarRepo(pool *pgxpool.Pool) *CalendarRepo { return &CalendarRepo{pool: pool} }

func (r *CalendarRepo) Get(ctx context.Context, calendarID string) (*domain.Calendar, error) {
	// Confirm calendar exists, then load holidays.
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM calendars WHERE calendar_id = $1)`, calendarID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("calendars.exists: %w", err)
	}
	if !exists {
		return nil, application.ErrNotFound
	}

	rows, err := r.pool.Query(ctx,
		`SELECT holiday_date FROM calendar_holidays WHERE calendar_id = $1 ORDER BY holiday_date`,
		calendarID,
	)
	if err != nil {
		return nil, fmt.Errorf("calendar_holidays.query: %w", err)
	}
	defer rows.Close()

	var holidays []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("calendar_holidays.scan: %w", err)
		}
		holidays = append(holidays, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("calendar_holidays.iter: %w", err)
	}
	return domain.ReconstituteCalendar(calendarID, holidays), nil
}

// ─── BICRepo ───────────────────────────────────────────────────────────────

type BICRepo struct{ pool *pgxpool.Pool }

func NewBICRepo(pool *pgxpool.Pool) *BICRepo { return &BICRepo{pool: pool} }

func (r *BICRepo) Resolve(ctx context.Context, bic string) (*domain.BICRecord, error) {
	var (
		name, country, lei string
		active             bool
	)
	err := r.pool.QueryRow(ctx,
		`SELECT institution_name, country, COALESCE(lei,''), active
		 FROM bic_records WHERE bic = $1`, bic,
	).Scan(&name, &country, &lei, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bic.resolve: %w", err)
	}
	return domain.ReconstituteBIC(bic, name, country, lei, active), nil
}

// ─── SSIRepo ───────────────────────────────────────────────────────────────

type SSIRepo struct{ pool *pgxpool.Pool }

func NewSSIRepo(pool *pgxpool.Pool) *SSIRepo { return &SSIRepo{pool: pool} }

func (r *SSIRepo) Find(ctx context.Context, tenantID uuid.UUID, cpBIC, currency string, atTime time.Time) (*domain.SSI, error) {
	var (
		id                            uuid.UUID
		benBIC, intBIC                string
		acct, iban                    string
		validFrom, validTo            time.Time
		hasValidTo                    bool
	)
	row := r.pool.QueryRow(ctx, `
		SELECT ssi_id,
		       beneficiary_bic,
		       COALESCE(intermediary_bic, ''),
		       COALESCE(account_number, ''),
		       COALESCE(iban, ''),
		       valid_from,
		       valid_to IS NOT NULL,
		       COALESCE(valid_to, valid_from)
		FROM ssis
		WHERE tenant_id = $1
		  AND counterparty_bic = $2
		  AND currency = $3
		  AND $4 >= valid_from
		  AND (valid_to IS NULL OR $4 <= valid_to)
		ORDER BY valid_from DESC
		LIMIT 1`,
		tenantID, cpBIC, currency, atTime,
	)
	if err := row.Scan(&id, &benBIC, &intBIC, &acct, &iban, &validFrom, &hasValidTo, &validTo); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, fmt.Errorf("ssi.find: %w", err)
	}
	if !hasValidTo {
		validTo = time.Time{}
	}
	return domain.ReconstituteSSI(id, tenantID, cpBIC, currency, benBIC, intBIC, acct, iban, validFrom, validTo), nil
}

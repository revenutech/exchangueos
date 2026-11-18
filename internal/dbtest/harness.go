//go:build integration

// Package dbtest — shared harness for integration tests that hit a live
// CockroachDB. Tests skip when EXCHANGEOS_TEST_DSN is not set so they remain
// no-ops on hosts without the docker-compose.test stack up.
//
// Bring the stack up with:
//
//	docker compose -f docker-compose.test.yml up -d crdb
//	migrate -path migrations -database "$EXCHANGEOS_TEST_DSN" up
//
// Then run:
//
//	EXCHANGEOS_TEST_DSN="postgres://root@localhost:26257/exchangeos?sslmode=disable" \
//	  go test -tags integration ./modules/...
package dbtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const envDSN = "EXCHANGEOS_TEST_DSN"

// PoolOrSkip returns a pgxpool.Pool against the test DSN, or skips the test
// when the env var is empty. The pool is closed via t.Cleanup.
func PoolOrSkip(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv(envDSN)
	if dsn == "" {
		t.Skipf("%s not set — skipping integration test", envDSN)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "open pool")
	require.NoError(t, pool.Ping(ctx), "ping crdb")
	t.Cleanup(pool.Close)
	return pool
}

// SeedTenant inserts a throwaway tenant + registers cleanup. Returns the id.
func SeedTenant(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	code := "test-" + id.String()[:8]
	_, err := pool.Exec(context.Background(), `
		INSERT INTO tenants (tenant_id, code, name, country, status)
		VALUES ($1, $2, $3, 'BR', 'ACTIVE')`,
		id, code, "Test "+code)
	require.NoError(t, err, "seed tenant")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM tenants WHERE tenant_id = $1`, id)
	})
	return id
}

// SeedCounterparty inserts a throwaway counterparty for the tenant. Returns id.
func SeedCounterparty(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	bic := "TEST" + id.String()[:7]
	_, err := pool.Exec(context.Background(), `
		INSERT INTO counterparties (counterparty_id, tenant_id, bic, name, country)
		VALUES ($1, $2, $3, $4, 'BR')`,
		id, tenantID, bic, "CP "+bic)
	require.NoError(t, err, "seed counterparty")
	return id
}

// SeedTrade inserts a throwaway fx_trade row for the tenant + returns its id.
// Seeds the buyer/seller counterparties on the fly.
func SeedTrade(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	buyer := SeedCounterparty(t, pool, tenantID)
	seller := SeedCounterparty(t, pool, tenantID)
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO fx_trades (
			trade_id, tenant_id, trade_type, settlement_venue,
			buyer_counterparty_id, seller_counterparty_id,
			bought_currency, bought_amount, sold_currency, sold_amount,
			deal_rate, trade_date, value_date
		) VALUES (
			$1, $2, 'SPOT', 'BILATERAL',
			$3, $4,
			'USD', 1000, 'BRL', 5000,
			5.0, current_timestamp(), current_date() + 2
		)`,
		id, tenantID, buyer, seller)
	require.NoError(t, err, "seed fx_trade")
	return id
}

// SeedCycle inserts a throwaway cls_cycles row + returns its id. Used by payin
// + netreport tests where the cycle is an FK target.
func SeedCycle(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO cls_cycles (
			cycle_id, tenant_id, cycle_date, status,
			opened_at, pin1_deadline, pin2_deadline, pin3_deadline,
			scheduled_close, version
		) VALUES (
			$1, $2, $3, 'OPEN',
			$4, $5, $6, $7,
			$8, 1
		)`,
		id, tenantID, day,
		now, now.Add(2*time.Hour), now.Add(4*time.Hour), now.Add(6*time.Hour),
		now.Add(8*time.Hour),
	)
	require.NoError(t, err, "seed cls_cycle")
	return id
}

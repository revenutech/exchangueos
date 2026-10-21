//go:build integration

// Integration tests for the compliance postgres repos. Skip when
// EXCHANGEOS_TEST_DSN is not set — see internal/dbtest/harness.go.
package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/internal/dbtest"
	"github.com/revenutech/exchangeos/modules/compliance/application"
	"github.com/revenutech/exchangeos/modules/compliance/domain"
	"github.com/revenutech/exchangeos/modules/compliance/infrastructure/postgres"
)

func TestClassificationRepo_RoundTrip(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewClassificationRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, err := domain.NewClassification(domain.NewClassificationInput{
		TenantID:    tenant,
		TradeID:     trade,
		Code:        "32101",
		Description: "Comércio exterior — Exportação",
		Nature:      domain.NatureIngresso,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, c))

	got, err := repo.GetByTrade(ctx, trade)
	require.NoError(t, err)
	assert.Equal(t, "32101", got.Code())
	assert.Equal(t, domain.NatureIngresso, got.Nature())
}

func TestClassificationRepo_ReplacesByTrade(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewClassificationRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	first, _ := domain.NewClassification(domain.NewClassificationInput{
		TenantID: tenant, TradeID: trade, Code: "32101",
		Description: "First", Nature: domain.NatureIngresso,
	})
	second, _ := domain.NewClassification(domain.NewClassificationInput{
		TenantID: tenant, TradeID: trade, Code: "63010",
		Description: "Second", Nature: domain.NatureRemessa,
	})
	require.NoError(t, repo.Save(ctx, first))
	require.NoError(t, repo.Save(ctx, second))

	got, err := repo.GetByTrade(ctx, trade)
	require.NoError(t, err)
	assert.Equal(t, "63010", got.Code(), "second Save must replace the first row")
}

func TestClassificationRepo_NotFound(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewClassificationRepo(pool)
	_, err := repo.GetByTrade(context.Background(), uuid.New())
	assert.ErrorIs(t, err, application.ErrNotFound)
}

func TestIOFRepo_RoundTrip(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewIOFRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	i, err := domain.NewIOFComputation(domain.NewIOFInput{
		TenantID:      tenant,
		TradeID:       trade,
		OperationType: "TRAVEL",
		Notional:      decimal.RequireFromString("1000.00"),
		NotionalCCY:   "USD",
		Rate:          decimal.RequireFromString("0.0038"),
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, i))

	got, err := repo.GetByTrade(ctx, trade)
	require.NoError(t, err)
	assert.Equal(t, "TRAVEL", got.OperationType())
	assert.True(t, got.IOFAmount().Equal(decimal.RequireFromString("3.80")),
		"computed iof must round-trip via decimal: got %s", got.IOFAmount().String())
}

func TestReportRepo_RoundTripAndTransitions(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewReportRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)

	r, err := domain.NewBACENReport(domain.NewBACENReportInput{
		TenantID:      tenant,
		ReportType:    domain.ReportSISBACEN,
		ReferenceDate: time.Now().UTC(),
		PayloadHash:   "abc123def456",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, r))

	got, err := repo.Get(ctx, r.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPending, got.Status())
	assert.Equal(t, 1, got.Version())

	require.NoError(t, r.MarkSubmitted(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, r))

	got, err = repo.Get(ctx, r.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSubmitted, got.Status())
	assert.Equal(t, 2, got.Version())
}

func TestReportRepo_VersionConflict(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewReportRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)

	r, _ := domain.NewBACENReport(domain.NewBACENReportInput{
		TenantID: tenant, ReportType: domain.ReportCambio,
		ReferenceDate: time.Now().UTC(), PayloadHash: "hash",
	})
	require.NoError(t, repo.Save(ctx, r))

	a, err := repo.Get(ctx, r.ID())
	require.NoError(t, err)
	require.NoError(t, a.MarkSubmitted(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, a))

	// `r` is still at version 1 → stale; second MarkSubmitted bumps to v2,
	// but DB already has v2. Save must reject.
	require.NoError(t, r.MarkSubmitted(time.Now().UTC()))
	err = repo.Save(ctx, r)
	assert.ErrorIs(t, err, application.ErrInvalidInput)
}

func TestScreeningRepo_AppendOnly(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewScreeningRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)

	low, _ := domain.NewScreeningResult(domain.NewScreeningInput{
		TenantID: tenant, CounterpartyBIC: "ABCDUSAA",
		Hits: nil,
	})
	high, _ := domain.NewScreeningResult(domain.NewScreeningInput{
		TenantID: tenant, CounterpartyBIC: "ABCDUSAA",
		Hits: []string{"OFAC:SDN:1", "UN:1267:2", "EU:3"},
	})
	require.NoError(t, repo.Save(ctx, low))
	require.NoError(t, repo.Save(ctx, high))

	// Both rows persist (append-only). Verify via raw query.
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM screening_results WHERE tenant_id = $1`, tenant,
	).Scan(&n))
	assert.Equal(t, 2, n)
}

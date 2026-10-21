//go:build integration

// Integration tests for the netreport postgres repo. Skip when
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
	"github.com/revenutech/exchangeos/modules/netreport/application"
	"github.com/revenutech/exchangeos/modules/netreport/domain"
	"github.com/revenutech/exchangeos/modules/netreport/infrastructure/postgres"
)

func newReport(t *testing.T, tenant, cycle uuid.UUID, ccy string, in, out string) *domain.NetReport {
	t.Helper()
	r, err := domain.NewNetReport(domain.NewNetReportInput{
		TenantID:    tenant,
		CycleID:     cycle,
		Currency:    ccy,
		GrossPayIn:  decimal.RequireFromString(in),
		GrossPayOut: decimal.RequireFromString(out),
		TradeCount:  4,
		GeneratedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	return r
}

func TestNetReportRepo_RoundTrip(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewNetReportRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	cycle := dbtest.SeedCycle(t, pool, tenant)

	r := newReport(t, tenant, cycle, "USD", "1000000", "600000")
	require.NoError(t, repo.Save(ctx, r))

	got, err := repo.GetByCycleCcy(ctx, cycle, "USD")
	require.NoError(t, err)
	assert.True(t, got.NetSettlement().Equal(decimal.RequireFromString("400000")),
		"net = 400000, got %s", got.NetSettlement().String())
	assert.Equal(t, 4, got.TradeCount())
}

func TestNetReportRepo_UpsertByCycleCurrency(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewNetReportRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	cycle := dbtest.SeedCycle(t, pool, tenant)

	first := newReport(t, tenant, cycle, "USD", "100", "100")
	require.NoError(t, repo.Save(ctx, first))

	// Regenerated with different gross values — should replace by
	// UNIQUE (cycle_id, currency).
	second := newReport(t, tenant, cycle, "USD", "500", "200")
	require.NoError(t, repo.Save(ctx, second))

	got, err := repo.GetByCycleCcy(ctx, cycle, "USD")
	require.NoError(t, err)
	assert.True(t, got.GrossPayIn().Equal(decimal.RequireFromString("500")))
	assert.True(t, got.GrossPayOut().Equal(decimal.RequireFromString("200")))
}

func TestNetReportRepo_ListByCycleOrderedByCcy(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewNetReportRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	cycle := dbtest.SeedCycle(t, pool, tenant)

	for _, ccy := range []string{"USD", "EUR", "JPY"} {
		require.NoError(t, repo.Save(ctx, newReport(t, tenant, cycle, ccy, "1000", "500")))
	}

	list, err := repo.ListByCycle(ctx, cycle)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "EUR", list[0].Currency())
	assert.Equal(t, "JPY", list[1].Currency())
	assert.Equal(t, "USD", list[2].Currency())
}

func TestNetReportRepo_NotFound(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewNetReportRepo(pool)
	_, err := repo.GetByCycleCcy(context.Background(), uuid.New(), "USD")
	assert.ErrorIs(t, err, application.ErrNotFound)
}

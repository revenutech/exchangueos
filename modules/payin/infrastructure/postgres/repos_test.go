//go:build integration

// Integration tests for the payin postgres repo. Skip when
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
	"github.com/revenutech/exchangeos/modules/payin/application"
	"github.com/revenutech/exchangeos/modules/payin/domain"
	"github.com/revenutech/exchangeos/modules/payin/infrastructure/postgres"
)

func newPending(t *testing.T, tenant, cycle uuid.UUID, ccy string) *domain.PayInInstruction {
	t.Helper()
	p, err := domain.NewPayInInstruction(domain.NewPayInInput{
		TenantID: tenant,
		CycleID:  cycle,
		Currency: ccy,
		Amount:   decimal.RequireFromString("1000000.00"),
		Band:     domain.BandPIN1,
		Deadline: time.Now().UTC().Add(2 * time.Hour),
	})
	require.NoError(t, err)
	return p
}

func TestPayInRepo_RoundTrip(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewPayInRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	cycle := dbtest.SeedCycle(t, pool, tenant)

	p := newPending(t, tenant, cycle, "USD")
	require.NoError(t, repo.Save(ctx, p))

	got, err := repo.Get(ctx, p.ID())
	require.NoError(t, err)
	assert.Equal(t, "USD", got.Currency())
	assert.Equal(t, domain.StatusPending, got.Status())
	assert.True(t, got.Amount().Equal(decimal.RequireFromString("1000000.00")))
}

func TestPayInRepo_ListByCycleSortedByCurrency(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewPayInRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	cycle := dbtest.SeedCycle(t, pool, tenant)

	for _, ccy := range []string{"USD", "EUR", "JPY"} {
		require.NoError(t, repo.Save(ctx, newPending(t, tenant, cycle, ccy)))
	}

	list, err := repo.ListByCycle(ctx, cycle)
	require.NoError(t, err)
	require.Len(t, list, 3)
	// Repo orders by currency ASC.
	assert.Equal(t, "EUR", list[0].Currency())
	assert.Equal(t, "JPY", list[1].Currency())
	assert.Equal(t, "USD", list[2].Currency())
}

func TestPayInRepo_TransitionAndVersionConflict(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewPayInRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	cycle := dbtest.SeedCycle(t, pool, tenant)

	p := newPending(t, tenant, cycle, "USD")
	require.NoError(t, repo.Save(ctx, p))

	a, err := repo.Get(ctx, p.ID())
	require.NoError(t, err)
	require.NoError(t, a.Submit(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, a))

	// `p` is stale; bumping to v2 must be rejected by the WHERE clause.
	require.NoError(t, p.Submit(time.Now().UTC()))
	err = repo.Save(ctx, p)
	assert.ErrorIs(t, err, application.ErrInvalidInput)
}

func TestPayInRepo_NotFound(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewPayInRepo(pool)
	_, err := repo.Get(context.Background(), uuid.New())
	assert.ErrorIs(t, err, application.ErrNotFound)
}

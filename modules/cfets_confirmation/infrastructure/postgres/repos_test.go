//go:build integration

// Integration tests for the cfets_confirmation postgres repo. Skip when
// EXCHANGEOS_TEST_DSN is not set — see internal/dbtest/harness.go.
package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/internal/dbtest"
	"github.com/revenutech/exchangeos/modules/cfets_confirmation/application"
	"github.com/revenutech/exchangeos/modules/cfets_confirmation/domain"
	"github.com/revenutech/exchangeos/modules/cfets_confirmation/infrastructure/postgres"
)

func TestCFETSConfirmationRepo_RoundTrip(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSConfirmationRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, err := domain.NewConfirmation(domain.NewConfirmationInput{
		TenantID:    tenant,
		TradeID:     trade,
		CFETSDealID: "DEAL-" + uuid.New().String()[:8],
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, c))

	got, err := repo.Get(ctx, c.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirming, got.Status())
	assert.Equal(t, 1, got.Version())
}

func TestCFETSConfirmationRepo_MarkPaired(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSConfirmationRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, _ := domain.NewConfirmation(domain.NewConfirmationInput{
		TenantID: tenant, TradeID: trade,
		CFETSDealID: "DEAL-" + uuid.New().String()[:8],
	})
	require.NoError(t, repo.Save(ctx, c))

	require.NoError(t, c.MarkPaired(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, c))

	got, err := repo.Get(ctx, c.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, got.Status())
	assert.Equal(t, 2, got.Version())
	assert.False(t, got.ConfirmedAt().IsZero())
}

func TestCFETSConfirmationRepo_VersionConflict(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSConfirmationRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, _ := domain.NewConfirmation(domain.NewConfirmationInput{
		TenantID: tenant, TradeID: trade,
		CFETSDealID: "DEAL-" + uuid.New().String()[:8],
	})
	require.NoError(t, repo.Save(ctx, c))

	a, err := repo.Get(ctx, c.ID())
	require.NoError(t, err)
	require.NoError(t, a.MarkPaired(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, a))

	require.NoError(t, c.MarkUnpaired(time.Now().UTC()))
	err = repo.Save(ctx, c)
	assert.ErrorIs(t, err, application.ErrInvalidInput)
}

func TestCFETSConfirmationRepo_NotFound(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSConfirmationRepo(pool)
	_, err := repo.Get(context.Background(), uuid.New())
	assert.ErrorIs(t, err, application.ErrNotFound)
}

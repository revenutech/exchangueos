//go:build integration

// Integration tests for the cfets_capture postgres repo. Skip when
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
	"github.com/revenutech/exchangeos/modules/cfets_capture/application"
	"github.com/revenutech/exchangeos/modules/cfets_capture/domain"
	"github.com/revenutech/exchangeos/modules/cfets_capture/infrastructure/postgres"
)

func TestCFETSCaptureRepo_RoundTrip(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSCaptureRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, err := domain.NewCapture(domain.NewCaptureInput{
		TenantID:     tenant,
		TradeID:      trade,
		SubmitterRef: "REF-" + uuid.New().String()[:8],
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, c))

	got, err := repo.Get(ctx, c.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDraft, got.Status())
	assert.Equal(t, 1, got.Version())
	assert.Equal(t, c.SubmitterRef(), got.SubmitterRef())
}

func TestCFETSCaptureRepo_TransitionsBumpVersion(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSCaptureRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, _ := domain.NewCapture(domain.NewCaptureInput{
		TenantID: tenant, TradeID: trade,
		SubmitterRef: "REF-" + uuid.New().String()[:8],
	})
	require.NoError(t, repo.Save(ctx, c))

	require.NoError(t, c.Submit(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, c))
	require.NoError(t, c.Ack(time.Now().UTC(), "CFETS-DEAL-001"))
	require.NoError(t, repo.Save(ctx, c))

	got, err := repo.Get(ctx, c.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusAck, got.Status())
	assert.Equal(t, "CFETS-DEAL-001", got.CFETSDealID())
	assert.Equal(t, 3, got.Version())
}

func TestCFETSCaptureRepo_VersionConflict(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSCaptureRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	trade := dbtest.SeedTrade(t, pool, tenant)

	c, _ := domain.NewCapture(domain.NewCaptureInput{
		TenantID: tenant, TradeID: trade,
		SubmitterRef: "REF-" + uuid.New().String()[:8],
	})
	require.NoError(t, repo.Save(ctx, c))

	// Writer A wins.
	a, err := repo.Get(ctx, c.ID())
	require.NoError(t, err)
	require.NoError(t, a.Submit(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, a))

	// Original `c` is stale; its Submit bumps in-memory to v2 but DB is at v2.
	require.NoError(t, c.Submit(time.Now().UTC()))
	err = repo.Save(ctx, c)
	assert.ErrorIs(t, err, application.ErrInvalidInput)
}

func TestCFETSCaptureRepo_NotFound(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewCFETSCaptureRepo(pool)
	_, err := repo.Get(context.Background(), uuid.New())
	assert.ErrorIs(t, err, application.ErrNotFound)
}

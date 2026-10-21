//go:build integration

// Integration tests for the admin postgres repos. Skip when
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
	"github.com/revenutech/exchangeos/modules/admin/application"
	"github.com/revenutech/exchangeos/modules/admin/domain"
	"github.com/revenutech/exchangeos/modules/admin/infrastructure/postgres"
)

func TestSystemEventRepo_InsertList(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewEventRepo(pool)
	ctx := context.Background()

	ev, err := domain.NewSystemEvent(domain.NewSystemEventInput{
		Code:        domain.EventStartup,
		Component:   "api",
		Description: "boot",
		At:          time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, ev))

	list, err := repo.List(ctx, 50)
	require.NoError(t, err)

	found := false
	for _, e := range list {
		if e.ID() == ev.ID() {
			found = true
			assert.Equal(t, domain.EventStartup, e.Code())
			assert.Equal(t, "api", e.Component())
			break
		}
	}
	assert.True(t, found, "expected just-saved event in list")
}

func TestSystemEventRepo_ListOrderedDesc(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewEventRepo(pool)
	ctx := context.Background()

	earlier, _ := domain.NewSystemEvent(domain.NewSystemEventInput{
		Code: domain.EventCycleOpen, Component: "cls", At: time.Now().UTC().Add(-2 * time.Hour),
	})
	later, _ := domain.NewSystemEvent(domain.NewSystemEventInput{
		Code: domain.EventCycleClose, Component: "cls", At: time.Now().UTC(),
	})
	require.NoError(t, repo.Save(ctx, earlier))
	require.NoError(t, repo.Save(ctx, later))

	list, err := repo.List(ctx, 5)
	require.NoError(t, err)
	require.NotEmpty(t, list)

	// Top of the list must be sorted by at DESC.
	for i := 1; i < len(list); i++ {
		assert.False(t, list[i].At().After(list[i-1].At()),
			"list must be sorted by at DESC")
	}
}

func TestEODJobRepo_RoundTripAndVersionBump(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewEODJobRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)

	job, err := domain.NewEODJob(domain.NewEODJobInput{
		TenantID:     tenant,
		BusinessDate: time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, job))

	got, err := repo.Get(ctx, job.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.EODPending, got.Status())
	assert.Equal(t, 1, got.Version())

	// Transition + Save — version bumps to 2.
	require.NoError(t, job.Start(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, job))

	got, err = repo.Get(ctx, job.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.EODRunning, got.Status())
	assert.Equal(t, 2, got.Version())
}

func TestEODJobRepo_VersionConflict(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewEODJobRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)

	job, _ := domain.NewEODJob(domain.NewEODJobInput{
		TenantID: tenant, BusinessDate: time.Now().UTC(),
	})
	require.NoError(t, repo.Save(ctx, job))

	// Loaded from DB: applies one transition + saves (writer A wins).
	a, err := repo.Get(ctx, job.ID())
	require.NoError(t, err)
	require.NoError(t, a.Start(time.Now().UTC()))
	require.NoError(t, repo.Save(ctx, a))

	// Original `job` in memory is stale (version 1, expecting to write 2);
	// but DB row is already at version 2 from writer A. Retrying its Save
	// must be rejected.
	err = repo.Save(ctx, job)
	assert.ErrorIs(t, err, application.ErrInvalidInput, "stale write must be rejected")
}

func TestEODJobRepo_FindByDate(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	repo := postgres.NewEODJobRepo(pool)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)

	target := time.Now().UTC().Add(-24 * time.Hour)
	job, _ := domain.NewEODJob(domain.NewEODJobInput{
		TenantID: tenant, BusinessDate: target,
	})
	require.NoError(t, repo.Save(ctx, job))

	got, err := repo.FindByDate(ctx, tenant, target)
	require.NoError(t, err)
	assert.Equal(t, job.ID(), got.ID())

	// Wrong tenant → ErrNotFound.
	_, err = repo.FindByDate(ctx, uuid.New(), target)
	assert.ErrorIs(t, err, application.ErrNotFound)
}

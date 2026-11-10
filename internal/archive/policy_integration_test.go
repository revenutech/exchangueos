//go:build integration

package archive_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/internal/archive"
	"github.com/revenutech/exchangeos/internal/dbtest"
)

// TestPgxStore_LoadsSeedAndCacheFollowsUpdates runs against migration 000010 (MS-024b 1.1/1.2): the 24 seeded
// policies load, and an UPDATE (with or without updated_at) reaches the cache on its next freshness check.
func TestPgxStore_LoadsSeedAndCacheFollowsUpdates(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	ctx := context.Background()
	store := archive.NewPgxStore(pool)

	all, err := store.LoadAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 24, "one policy per archivable table of data-lifecycle/README.md")

	cache, err := archive.NewCache(store, archive.WithCheckEvery(time.Nanosecond))
	require.NoError(t, err)

	q, ok, err := cache.Get(ctx, "quotes")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, archive.Interval{Days: 90}, q.AgeThreshold)
	require.NotNil(t, q.RetentionAfter)
	assert.Equal(t, archive.Interval{Months: 60}, *q.RetentionAfter, "5 years = 60 months")
	assert.False(t, q.LegalHold)

	f, ok, err := cache.Get(ctx, "fx_trades")
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, f.LegalHold)
	assert.Nil(t, f.RetentionAfter)

	// UPDATE with updated_at
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE archive_policy SET age_threshold = INTERVAL '90 days', gcs_prefix = 'quotes/', updated_at = now() WHERE table_name = 'quotes'`)
	})
	_, err = pool.Exec(ctx, `UPDATE archive_policy SET age_threshold = INTERVAL '45 days', updated_at = now() WHERE table_name = 'quotes'`)
	require.NoError(t, err)
	q, _, err = cache.Get(ctx, "quotes")
	require.NoError(t, err)
	assert.Equal(t, archive.Interval{Days: 45}, q.AgeThreshold)

	// UPDATE that leaves updated_at alone: the digest still moves
	_, err = pool.Exec(ctx, `UPDATE archive_policy SET gcs_prefix = 'quotes_v2/' WHERE table_name = 'quotes'`)
	require.NoError(t, err)
	q, _, err = cache.Get(ctx, "quotes")
	require.NoError(t, err)
	assert.Equal(t, "quotes_v2/", q.GCSPrefix)
}

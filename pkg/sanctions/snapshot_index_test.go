package sanctions

import (
	"io/fs"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testClock is a concurrency-safe clock the test moves by hand.
type testClock struct{ offset atomic.Int64 }

func (c *testClock) now() time.Time          { return snapTime.Add(time.Duration(c.offset.Load())) }
func (c *testClock) advance(d time.Duration) { c.offset.Add(int64(d)) }
func (c *testClock) set(d time.Duration)     { c.offset.Store(int64(d)) }
func freshSnap(fetchedAt time.Time) Snapshot {
	return Snapshot{Source: SourceOFAC, FetchedAt: fetchedAt, Entries: snapEntries()}
}
func shipQuery() MatchQuery { return MatchQuery{Name: shipName, MinScore: 0.9} }
func newTestSnapshotIndex(c *testClock) *SnapshotIndex {
	return NewSnapshotIndex(SourceOFAC, MaxSnapshotAge, c.now)
}

func TestSnapshotAge(t *testing.T) {
	t.Parallel()
	s := freshSnap(snapTime)
	assert.Equal(t, 90*time.Minute, s.Age(snapTime.Add(90*time.Minute)))
	assert.Equal(t, 24*time.Hour, MaxSnapshotAge)
}

func TestSnapshotIndex_BlocksBeforeFirstLoad(t *testing.T) {
	t.Parallel()
	idx := newTestSnapshotIndex(&testClock{})
	_, err := idx.Match(shipQuery())
	require.ErrorIs(t, err, ErrStaleSnapshot)
}

func TestSnapshotIndex_MatchWithinMaxAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		elapsed time.Duration
		blocked bool
	}{
		{"just loaded", 0, false},
		{"one hour", time.Hour, false},
		{"exactly 24h", MaxSnapshotAge, false},
		{"24h and a second", MaxSnapshotAge + time.Second, true},
		{"three days", 72 * time.Hour, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clk := &testClock{}
			idx := newTestSnapshotIndex(clk)
			require.NoError(t, idx.Load(freshSnap(snapTime)))
			clk.set(tt.elapsed)
			hits, err := idx.Match(shipQuery())
			if tt.blocked {
				require.ErrorIs(t, err, ErrStaleSnapshot)
				assert.Nil(t, hits)
				assert.NotContains(t, err.Error(), "Mardok", "the query name is PII and never reaches the error")
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, hits)
			assert.Equal(t, "990002", hits[0].EntryID)
		})
	}
}

func TestSnapshotIndex_LoadRefusesStaleAndInvalid(t *testing.T) {
	t.Parallel()
	other := freshSnap(snapTime)
	other.Source = SourceUN
	for i := range other.Entries {
		other.Entries[i].Source = SourceUN
	}
	tests := []struct {
		name string
		snap Snapshot
		want error
	}{
		{"older than max age", freshSnap(snapTime.Add(-MaxSnapshotAge - time.Minute)), ErrStaleSnapshot},
		{"no entries", Snapshot{Source: SourceOFAC, FetchedAt: snapTime}, ErrInvalidSnapshot},
		{"another list", other, ErrInvalidSnapshot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clk := &testClock{}
			idx := newTestSnapshotIndex(clk)
			// A fresh snapshot is serving; the refused load leaves it in place.
			require.NoError(t, idx.Load(freshSnap(snapTime.Add(-time.Hour))))
			require.ErrorIs(t, idx.Load(tt.snap), tt.want)
			_, err := idx.Match(shipQuery())
			require.NoError(t, err)
			assert.True(t, snapTime.Add(-time.Hour).Equal(idx.FetchedAt()))
		})
	}
}

// The upstream being down only means the snapshot file is not replaced: Reload failing (missing or broken file)
// keeps the loaded snapshot serving until it is older than MaxSnapshotAge, then Match blocks.
func TestSnapshotIndex_ReloadFailureKeepsServingUntilMaxAge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	clk := &testClock{}
	idx := newTestSnapshotIndex(clk)
	require.NoError(t, WriteSnapshot(dir, freshSnap(snapTime)))
	require.NoError(t, idx.Reload(dir))

	empty := t.TempDir()
	clk.advance(12 * time.Hour)
	require.ErrorIs(t, idx.Reload(empty), fs.ErrNotExist)
	_, err := idx.Match(shipQuery())
	require.NoError(t, err, "a failed reload must not stop Match")

	clk.advance(13 * time.Hour)
	_, err = idx.Match(shipQuery())
	require.ErrorIs(t, err, ErrStaleSnapshot)

	// The same old file reloaded does not unblock: it is still older than the limit.
	require.ErrorIs(t, idx.Reload(dir), ErrStaleSnapshot)
	_, err = idx.Match(shipQuery())
	require.ErrorIs(t, err, ErrStaleSnapshot)

	// A new snapshot does.
	require.NoError(t, WriteSnapshot(dir, freshSnap(clk.now())))
	require.NoError(t, idx.Reload(dir))
	_, err = idx.Match(shipQuery())
	require.NoError(t, err)
}

func TestSnapshotIndex_InvalidQuery(t *testing.T) {
	t.Parallel()
	idx := newTestSnapshotIndex(&testClock{})
	require.NoError(t, idx.Load(freshSnap(snapTime)))
	_, err := idx.Match(MatchQuery{Name: " "})
	require.ErrorIs(t, err, ErrInvalidQuery)
}

func TestSnapshotIndex_ConcurrentLoadAndMatch(t *testing.T) {
	t.Parallel()
	clk := &testClock{}
	idx := newTestSnapshotIndex(clk)
	require.NoError(t, idx.Load(freshSnap(snapTime)))
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 50 {
				_ = idx.Load(freshSnap(snapTime))
			}
		}()
		go func() {
			defer wg.Done()
			for range 50 {
				_, err := idx.Match(shipQuery())
				assert.NoError(t, err)
			}
		}()
	}
	wg.Wait()
}

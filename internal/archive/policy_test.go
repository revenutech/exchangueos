package archive_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/internal/archive"
)

// fakeStore counts calls and serves whatever the test puts in it.
type fakeStore struct {
	mu        sync.Mutex
	policies  []archive.Policy
	fp        archive.Fingerprint
	loadErr   error
	fpErr     error
	loads     int
	fpCalls   int
}

func (f *fakeStore) LoadAll(context.Context) ([]archive.Policy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loads++
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return append([]archive.Policy(nil), f.policies...), nil
}

func (f *fakeStore) Fingerprint(context.Context) (archive.Fingerprint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fpCalls++
	if f.fpErr != nil {
		return archive.Fingerprint{}, f.fpErr
	}
	return f.fp, nil
}

func (f *fakeStore) set(fn func(f *fakeStore)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

func (f *fakeStore) counts() (loads, fps int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.loads, f.fpCalls
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var t0 = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

func years(n int32) archive.Interval { return archive.Interval{Months: 12 * n} }
func days(n int32) archive.Interval  { return archive.Interval{Days: n} }
func ptr(i archive.Interval) *archive.Interval { return &i }

func seed() []archive.Policy {
	return []archive.Policy{
		{TableName: "quotes", AgeThreshold: days(90), RetentionAfter: ptr(years(5)), GCSPrefix: "quotes/", UpdatedAt: t0},
		{TableName: "fx_trades", AgeThreshold: years(1), GCSPrefix: "fx_trades/", LegalHold: true, UpdatedAt: t0},
	}
}

func newCache(t *testing.T, st archive.Store, c *clock) *archive.Cache {
	t.Helper()
	cache, err := archive.NewCache(st, archive.WithCheckEvery(30*time.Second), archive.WithClock(c.Now))
	require.NoError(t, err)
	return cache
}

func TestNewCache_RejectsBadInput(t *testing.T) {
	t.Parallel()
	_, err := archive.NewCache(nil)
	require.Error(t, err)
	_, err = archive.NewCache(&fakeStore{}, archive.WithCheckEvery(0))
	require.Error(t, err)
	_, err = archive.NewCache(&fakeStore{}, archive.WithClock(nil))
	require.Error(t, err)
}

func TestCache_LoadsOnFirstUseAndServesAll(t *testing.T) {
	t.Parallel()
	st := &fakeStore{policies: seed(), fp: archive.Fingerprint{Count: 2, MaxUpdatedAt: t0, Digest: "a"}}
	c := &clock{now: t0}
	cache := newCache(t, st, c)

	all, err := cache.All(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "fx_trades", all[0].TableName, "sorted by table name")
	assert.Equal(t, "quotes", all[1].TableName)

	p, ok, err := cache.Get(context.Background(), "quotes")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, days(90), p.AgeThreshold)

	_, ok, err = cache.Get(context.Background(), "currencies")
	require.NoError(t, err)
	assert.False(t, ok, "a «Never» table has no policy")

	loads, _ := st.counts()
	assert.Equal(t, 1, loads, "second read is a cache hit")
}

func TestCache_ReturnsCopies(t *testing.T) {
	t.Parallel()
	st := &fakeStore{policies: seed(), fp: archive.Fingerprint{Count: 2}}
	cache := newCache(t, st, &clock{now: t0})
	all, err := cache.All(context.Background())
	require.NoError(t, err)
	all[0].TableName = "mutated"
	*all[1].RetentionAfter = days(1)

	again, err := cache.All(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fx_trades", again[0].TableName)
	assert.Equal(t, years(5), *again[1].RetentionAfter)
}

func TestCache_ChecksFingerprintAtMostEveryInterval(t *testing.T) {
	t.Parallel()
	st := &fakeStore{policies: seed(), fp: archive.Fingerprint{Count: 2, Digest: "a"}}
	c := &clock{now: t0}
	cache := newCache(t, st, c)
	ctx := context.Background()

	_, err := cache.All(ctx)
	require.NoError(t, err)
	_, fp0 := st.counts()

	c.advance(29 * time.Second)
	_, err = cache.All(ctx)
	require.NoError(t, err)
	_, fp1 := st.counts()
	assert.Equal(t, fp0, fp1, "inside the interval: no fingerprint query")

	c.advance(time.Second)
	_, err = cache.All(ctx)
	require.NoError(t, err)
	loads, fp2 := st.counts()
	assert.Equal(t, fp1+1, fp2, "interval elapsed: one fingerprint query")
	assert.Equal(t, 1, loads, "fingerprint unchanged: no reload")
}

func TestCache_ReloadsWhenARowIsUpdated(t *testing.T) {
	t.Parallel()
	st := &fakeStore{policies: seed(), fp: archive.Fingerprint{Count: 2, MaxUpdatedAt: t0, Digest: "a"}}
	c := &clock{now: t0}
	cache := newCache(t, st, c)
	ctx := context.Background()
	_, err := cache.All(ctx)
	require.NoError(t, err)

	// UPDATE archive_policy SET age_threshold = '45 days' WHERE table_name = 'quotes'
	st.set(func(f *fakeStore) {
		f.policies[0].AgeThreshold = days(45)
		f.fp = archive.Fingerprint{Count: 2, MaxUpdatedAt: t0, Digest: "b"} // updated_at untouched: the digest still moves
	})
	c.advance(30 * time.Second)

	p, ok, err := cache.Get(ctx, "quotes")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, days(45), p.AgeThreshold)
	loads, _ := st.counts()
	assert.Equal(t, 2, loads)
}

func TestCache_ErrorsFailClosed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	boom := errors.New("crdb down")

	st := &fakeStore{loadErr: boom}
	cache := newCache(t, st, &clock{now: t0})
	_, err := cache.All(ctx)
	require.ErrorIs(t, err, boom, "first load fails")

	st2 := &fakeStore{policies: seed(), fp: archive.Fingerprint{Count: 2, Digest: "a"}}
	c := &clock{now: t0}
	cache2 := newCache(t, st2, c)
	_, err = cache2.All(ctx)
	require.NoError(t, err)
	st2.set(func(f *fakeStore) { f.fpErr = boom })
	c.advance(time.Minute)
	_, _, err = cache2.Get(ctx, "quotes")
	require.ErrorIs(t, err, boom, "a failed freshness check is an error, not a stale answer: the archiver deletes by policy")

	st2.set(func(f *fakeStore) { f.fpErr = nil; f.fp = archive.Fingerprint{Count: 3, Digest: "c"}; f.loadErr = boom })
	_, err = cache2.All(ctx)
	require.ErrorIs(t, err, boom, "reload failure surfaces")
}

func TestCache_RejectsBadPolicies(t *testing.T) {
	t.Parallel()
	bad := seed()
	bad = append(bad, bad[0]) // duplicate table
	st := &fakeStore{policies: bad, fp: archive.Fingerprint{Count: 3}}
	_, err := newCache(t, st, &clock{now: t0}).All(context.Background())
	require.Error(t, err)

	held := seed()
	held[1].RetentionAfter = ptr(years(10)) // legal hold with a retention
	st = &fakeStore{policies: held, fp: archive.Fingerprint{Count: 2}}
	_, err = newCache(t, st, &clock{now: t0}).All(context.Background())
	require.Error(t, err)
}

func TestCache_ConcurrentReaders(t *testing.T) {
	t.Parallel()
	st := &fakeStore{policies: seed(), fp: archive.Fingerprint{Count: 2, Digest: "a"}}
	c := &clock{now: t0}
	cache := newCache(t, st, c)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if i == 0 && j%10 == 0 {
					c.advance(time.Minute)
					st.set(func(f *fakeStore) { f.fp.Digest = time.Now().String() })
				}
				_, _, err := cache.Get(ctx, "quotes")
				assert.NoError(t, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestInterval_CutoffKeepsCalendarMonthsAndDays(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   archive.Interval
		want time.Time
	}{
		{"1 year is 12 calendar months", years(1), time.Date(2025, 3, 31, 10, 0, 0, 0, time.UTC)},
		{"90 days are days, not 3 months", days(90), time.Date(2025, 12, 31, 10, 0, 0, 0, time.UTC)},
		{"7 years across a leap day", years(7), time.Date(2019, 3, 31, 10, 0, 0, 0, time.UTC)},
		{"sub-day part", archive.Interval{Days: 1, Microseconds: int64(time.Hour / time.Microsecond)}, time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.in.Before(now))
		})
	}
}

func TestPolicy_Cutoffs(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	p := seed()[0]
	assert.Equal(t, time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), p.AgeCutoff(now))
	rc, ok := p.RetentionCutoff(now)
	require.True(t, ok)
	assert.Equal(t, time.Date(2021, 11, 10, 0, 0, 0, 0, time.UTC), rc)

	held := seed()[1]
	_, ok = held.RetentionCutoff(now)
	assert.False(t, ok, "legal hold: never hard-deleted")
}

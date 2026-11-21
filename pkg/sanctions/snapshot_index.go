package sanctions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// MaxSnapshotAge is the staleness limit of a list (MS-024d 1.9, RTSK.024.1209.01): a snapshot whose download
// started more than 24 hours ago no longer screens anyone. It is the same 24 hours as the freshness SLO and its
// alert (sanctions_list_age_seconds > 86400).
const MaxSnapshotAge = 24 * time.Hour

// ErrStaleSnapshot is returned when the snapshot of a list is older than its limit, or when there is none: screening
// against that list is blocked until a fresh snapshot arrives.
var ErrStaleSnapshot = errors.New("sanctions: snapshot older than the staleness limit")

// Age is how long ago, at now, the download that produced the snapshot started.
func (s Snapshot) Age(now time.Time) time.Duration { return now.Sub(s.FetchedAt) }

// SnapshotIndex is the matcher a screening pod holds for one list: a Cache loaded from the refresher's snapshot
// file, and the fetched_at of that snapshot.
//
// The pod never talks to the upstream, so an upstream outage cannot stop Match: the refresher keeps the last good
// file, and a Reload that fails (file missing, unreadable, invalid) keeps the loaded snapshot serving. What stops
// Match is age: once the loaded snapshot is older than maxAge, Match fails with ErrStaleSnapshot, and a stale file
// is refused by Load. Screening against an outdated list fails closed instead of passing names a newer list
// would catch.
type SnapshotIndex struct {
	source Source
	maxAge time.Duration
	now    func() time.Time
	cache  *Cache

	mu        sync.RWMutex // guards fetchedAt; the cache swaps its own content atomically
	fetchedAt time.Time    // zero until the first Load
}

var _ contextMatcher = (*SnapshotIndex)(nil)

// NewSnapshotIndex returns an empty index for source; Match is blocked until the first Load. A non-positive maxAge
// means MaxSnapshotAge, a nil now means time.Now. opts configure the underlying Cache: in production,
// WithHitAuditor, so every screening's hits are audited (MS-024d 1.11) under the request's HitScope.
func NewSnapshotIndex(source Source, maxAge time.Duration, now func() time.Time, opts ...CacheOption) *SnapshotIndex {
	if maxAge <= 0 {
		maxAge = MaxSnapshotAge
	}
	if now == nil {
		now = time.Now
	}
	return &SnapshotIndex{source: source, maxAge: maxAge, now: now, cache: NewCache(opts...)}
}

// FetchedAt reports the fetched_at of the loaded snapshot (zero before the first Load).
func (x *SnapshotIndex) FetchedAt() time.Time {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.fetchedAt
}

// Load swaps snap in. A snapshot of another list, an invalid one, or one already older than maxAge is refused and the
// loaded snapshot keeps serving.
func (x *SnapshotIndex) Load(snap Snapshot) error {
	if snap.Source != x.source {
		return fmt.Errorf("%w: %s index offered a snapshot of %q", ErrInvalidSnapshot, x.source, snap.Source)
	}
	if err := snap.validate(); err != nil {
		return err
	}
	if age := snap.Age(x.now()); age > x.maxAge {
		return fmt.Errorf("%w: %s snapshot is %s old (limit %s)", ErrStaleSnapshot, x.source, age.Round(time.Second), x.maxAge)
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.cache.Load(snap.Entries); err != nil {
		return fmt.Errorf("sanctions: load %s snapshot: %w", x.source, err)
	}
	x.fetchedAt = snap.FetchedAt
	return nil
}

// Reload reads the list's snapshot file from dir (ReadSnapshot) and Loads it. On any error the loaded snapshot keeps
// serving.
func (x *SnapshotIndex) Reload(dir string) error {
	snap, err := ReadSnapshot(dir, x.source)
	if err != nil {
		return err
	}
	return x.Load(snap)
}

// Match is MatchContext without a context. With an auditor wired it fails with ErrHitAudit (no tenant), so it only
// serves the no-op default.
func (x *SnapshotIndex) Match(q MatchQuery) ([]Match, error) {
	return x.MatchContext(context.Background(), q)
}

// MatchContext screens q against the loaded snapshot. It fails with ErrStaleSnapshot when there is no snapshot or it
// is older than maxAge; that check comes first, so a stale list blocks before any audit. Otherwise it is
// Cache.MatchContext: with an auditor, ctx must carry a HitScope with a tenant (ErrHitAudit otherwise), and the hits
// are audited under it. Errors name the list and the age, never the query.
func (x *SnapshotIndex) MatchContext(ctx context.Context, q MatchQuery) ([]Match, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	x.mu.RLock()
	fetchedAt := x.fetchedAt
	x.mu.RUnlock()
	if fetchedAt.IsZero() {
		return nil, fmt.Errorf("%w: no %s snapshot loaded", ErrStaleSnapshot, x.source)
	}
	if age := x.now().Sub(fetchedAt); age > x.maxAge {
		return nil, fmt.Errorf("%w: %s snapshot is %s old (limit %s)", ErrStaleSnapshot, x.source, age.Round(time.Second), x.maxAge)
	}
	return x.cache.MatchContext(ctx, q)
}

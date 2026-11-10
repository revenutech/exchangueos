// Package archive ages CockroachDB rows out to GCS per the data-driven archive_policy table
// (docs/security/data-lifecycle/archival-cron.md, MS-024b).
//
// This file is the policy loader and cache (MS-024b 1.2, RTSK.024.1002.01): migration 000010 seeds one policy per
// table with an archival rule (24); operators may UPDATE them, and the cache picks the change up on its next
// freshness check without a restart.
package archive

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Interval mirrors a SQL INTERVAL without flattening it. Convention: years are 12 months (CockroachDB stores
// '5 years' as 60 months) and months and days stay calendar units, applied with time.AddDate; only the sub-day part
// is a fixed duration. A flattened time.Duration would turn «1 year» into 365 days and drift on leap years, and
// «1 month» into 30 days; the archiver decides deletion by these cutoffs, so the calendar meaning is kept.
type Interval struct {
	Months       int32
	Days         int32
	Microseconds int64
}

// Before returns t minus the interval: months and days on the calendar, then the sub-day part.
func (i Interval) Before(t time.Time) time.Time {
	return t.AddDate(0, -int(i.Months), -int(i.Days)).Add(-time.Duration(i.Microseconds) * time.Microsecond)
}

// IsPositive reports whether the interval moves time forward on every component that is set.
func (i Interval) IsPositive() bool {
	return i.Months >= 0 && i.Days >= 0 && i.Microseconds >= 0 && (i.Months > 0 || i.Days > 0 || i.Microseconds > 0)
}

// Policy is one row of archive_policy.
type Policy struct {
	TableName      string
	AgeThreshold   Interval  // archive rows older than this
	RetentionAfter *Interval // hard-delete from CRDB after this; nil = keep (legal hold)
	GCSPrefix      string
	LegalHold      bool
	UpdatedAt      time.Time
}

// AgeCutoff is the instant before which rows are old enough to archive.
func (p Policy) AgeCutoff(now time.Time) time.Time { return p.AgeThreshold.Before(now) }

// RetentionCutoff is the instant before which archived rows may be hard-deleted; false when the table is never
// hard-deleted (legal hold, or no retention).
func (p Policy) RetentionCutoff(now time.Time) (time.Time, bool) {
	if p.LegalHold || p.RetentionAfter == nil {
		return time.Time{}, false
	}
	return p.RetentionAfter.Before(now), true
}

func (p Policy) clone() Policy {
	if p.RetentionAfter != nil {
		r := *p.RetentionAfter
		p.RetentionAfter = &r
	}
	return p
}

// Fingerprint is a cheap summary of the whole table. Digest covers every column but updated_at, so an UPDATE that
// forgets to bump updated_at still changes it.
type Fingerprint struct {
	Count        int
	MaxUpdatedAt time.Time
	Digest       string
}

// Store reads archive_policy.
type Store interface {
	LoadAll(ctx context.Context) ([]Policy, error)
	Fingerprint(ctx context.Context) (Fingerprint, error)
}

// Querier is the part of *pgxpool.Pool (or pgx.Conn / pgx.Tx) the store needs.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PgxStore is the CockroachDB-backed Store.
type PgxStore struct{ q Querier }

// NewPgxStore wraps a pool, connection or transaction.
func NewPgxStore(q Querier) *PgxStore { return &PgxStore{q: q} }

const loadSQL = `SELECT table_name, age_threshold, retention_after, gcs_prefix, legal_hold, updated_at
FROM archive_policy ORDER BY table_name`

const fingerprintSQL = `SELECT count(*), max(updated_at),
  coalesce(md5(string_agg(
    table_name || '|' || age_threshold::STRING || '|' || coalesce(retention_after::STRING, '-') || '|' ||
    gcs_prefix || '|' || legal_hold::STRING, ',' ORDER BY table_name)), '')
FROM archive_policy`

// LoadAll reads every policy.
func (s *PgxStore) LoadAll(ctx context.Context) ([]Policy, error) {
	rows, err := s.q.Query(ctx, loadSQL)
	if err != nil {
		return nil, fmt.Errorf("archive_policy: query: %w", err)
	}
	defer rows.Close()
	var out []Policy
	for rows.Next() {
		var (
			p        Policy
			age, ret pgtype.Interval
		)
		if err := rows.Scan(&p.TableName, &age, &ret, &p.GCSPrefix, &p.LegalHold, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("archive_policy: scan: %w", err)
		}
		if !age.Valid {
			return nil, fmt.Errorf("archive_policy: %s: age_threshold is NULL", p.TableName)
		}
		p.AgeThreshold = fromPg(age)
		if ret.Valid {
			r := fromPg(ret)
			p.RetentionAfter = &r
		}
		p.UpdatedAt = p.UpdatedAt.UTC()
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("archive_policy: rows: %w", err)
	}
	return out, nil
}

// Fingerprint summarises the table in one row.
func (s *PgxStore) Fingerprint(ctx context.Context) (Fingerprint, error) {
	var (
		fp  Fingerprint
		max pgtype.Timestamptz
	)
	if err := s.q.QueryRow(ctx, fingerprintSQL).Scan(&fp.Count, &max, &fp.Digest); err != nil {
		return Fingerprint{}, fmt.Errorf("archive_policy: fingerprint: %w", err)
	}
	if max.Valid {
		fp.MaxUpdatedAt = max.Time.UTC()
	}
	return fp, nil
}

func fromPg(i pgtype.Interval) Interval {
	return Interval{Months: i.Months, Days: i.Days, Microseconds: i.Microseconds}
}

// Cache serves the policies from memory and reloads them when the table's fingerprint changes. The fingerprint is
// queried at most once per check interval. A failed load or freshness check is returned as an error, never answered
// from a possibly stale copy: the archiver deletes by these policies, so it must fail closed.
type Cache struct {
	store Store
	now   func() time.Time
	every time.Duration

	mu        sync.Mutex
	loaded    bool
	fp        Fingerprint
	checkedAt time.Time
	byTable   map[string]Policy
	sorted    []Policy
}

// Option configures a Cache.
type Option func(*Cache)

// WithCheckEvery sets the minimum interval between fingerprint queries (default 30 s).
func WithCheckEvery(d time.Duration) Option { return func(c *Cache) { c.every = d } }

// WithClock injects the clock (tests).
func WithClock(now func() time.Time) Option { return func(c *Cache) { c.now = now } }

// NewCache builds a cache over store; nothing is loaded until the first read.
func NewCache(store Store, opts ...Option) (*Cache, error) {
	if store == nil {
		return nil, errors.New("archive: nil store")
	}
	c := &Cache{store: store, now: time.Now, every: 30 * time.Second}
	for _, o := range opts {
		o(c)
	}
	if c.every <= 0 {
		return nil, fmt.Errorf("archive: check interval must be positive, got %s", c.every)
	}
	if c.now == nil {
		return nil, errors.New("archive: nil clock")
	}
	return c, nil
}

// All returns every policy, sorted by table name. The slice and the policies are copies.
func (c *Cache) All(ctx context.Context) ([]Policy, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.freshLocked(ctx); err != nil {
		return nil, err
	}
	out := make([]Policy, len(c.sorted))
	for i, p := range c.sorted {
		out[i] = p.clone()
	}
	return out, nil
}

// Get returns the policy of one table; false when the table has none (the «Never» tables).
func (c *Cache) Get(ctx context.Context, table string) (Policy, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.freshLocked(ctx); err != nil {
		return Policy{}, false, err
	}
	p, ok := c.byTable[table]
	return p.clone(), ok, nil
}

func (c *Cache) freshLocked(ctx context.Context) error {
	now := c.now()
	if c.loaded && now.Sub(c.checkedAt) < c.every {
		return nil
	}
	fp, err := c.store.Fingerprint(ctx)
	if err != nil {
		return err
	}
	if c.loaded && fp == c.fp {
		c.checkedAt = now
		return nil
	}
	// The fingerprint is read before the rows: an UPDATE landing in between makes the next check reload again.
	policies, err := c.store.LoadAll(ctx)
	if err != nil {
		return err
	}
	byTable := make(map[string]Policy, len(policies))
	for _, p := range policies {
		if err := validate(p); err != nil {
			return err
		}
		if _, dup := byTable[p.TableName]; dup {
			return fmt.Errorf("archive: duplicate policy for table %q", p.TableName)
		}
		byTable[p.TableName] = p.clone()
	}
	sorted := make([]Policy, 0, len(byTable))
	for _, p := range byTable {
		sorted = append(sorted, p)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].TableName < sorted[j].TableName })
	c.byTable, c.sorted, c.fp, c.checkedAt, c.loaded = byTable, sorted, fp, now, true
	return nil
}

// validate repeats the table's CHECKs, so a policy that reached the cache some other way is refused too.
func validate(p Policy) error {
	if p.TableName == "" {
		return errors.New("archive: policy without table name")
	}
	if !p.AgeThreshold.IsPositive() {
		return fmt.Errorf("archive: %s: age_threshold must be positive", p.TableName)
	}
	if p.RetentionAfter != nil && !p.RetentionAfter.IsPositive() {
		return fmt.Errorf("archive: %s: retention_after must be positive", p.TableName)
	}
	if p.LegalHold && p.RetentionAfter != nil {
		return fmt.Errorf("archive: %s: legal hold with a retention", p.TableName)
	}
	if p.GCSPrefix == "" {
		return fmt.Errorf("archive: %s: empty gcs prefix", p.TableName)
	}
	return nil
}

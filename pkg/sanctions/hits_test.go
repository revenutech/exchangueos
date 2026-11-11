package sanctions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingAuditor keeps every batch it is given.
type recordingAuditor struct {
	mu      sync.Mutex
	calls   int
	scopes  []HitScope
	hits    []Hit
	failure error
}

func (r *recordingAuditor) RecordHits(_ context.Context, scope HitScope, hits []Hit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.scopes = append(r.scopes, scope)
	r.hits = append(r.hits, hits...)
	return r.failure
}

func (r *recordingAuditor) snapshot() (int, []HitScope, []Hit) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, append([]HitScope(nil), r.scopes...), append([]Hit(nil), r.hits...)
}

func hitIDs(hs []Hit) []string {
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.EntryID
	}
	return out
}

// queryName is synthetic and deliberately different from every list name below, so a leak is attributable to the
// query and not to the (public) list data.
const queryName = "Synthetik Tradinq"

func auditedCache(t *testing.T, a HitAuditor) *Cache {
	t.Helper()
	c := NewCache(WithHitAuditor(a))
	require.NoError(t, c.Load([]Entry{
		ent("a", "Synthetic Trading"),
		ent("d", "Synthetic Banana Republic"),
		ent("z", "Unrelated Person"),
	}))
	return c
}

func TestCache_AuditsEveryScoredHitIncludingSubThreshold(t *testing.T) {
	t.Parallel()
	rec := &recordingAuditor{}
	c := auditedCache(t, rec)

	got, err := c.Match(MatchQuery{Name: queryName, MinScore: 0.9})
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, ids(got), "the threshold still filters what Match returns")

	calls, _, hits := rec.snapshot()
	require.Equal(t, 1, calls, "one batch per Match")
	require.Equal(t, []string{"a", "d"}, hitIDs(hits), "every scored entry is audited, best first; «z» never scored")

	above, below := hits[0], hits[1]
	assert.Equal(t, "SANCTIONS_HIT", HitEventType)
	assert.Equal(t, SourceOFAC, above.Source)
	assert.True(t, above.AboveThreshold)
	assert.InDelta(t, got[0].Score, above.Score, 1e-12, "the audited score is the returned score")
	assert.Equal(t, 0.9, above.MinScore)
	assert.Equal(t, SourceOFAC, below.Source)
	assert.False(t, below.AboveThreshold, "sub-threshold hits are audited too")
	assert.Less(t, below.Score, 0.9)
	assert.Equal(t, 0.9, below.MinScore)
}

func TestCache_AuditCarriesScopeFromContext(t *testing.T) {
	t.Parallel()
	scope := HitScope{TenantID: "00000000-0000-4000-8000-000000000001", CorrelationID: "corr-synthetic-1"}
	cases := []struct {
		name  string
		ctx   context.Context
		match func(ctx context.Context, t *testing.T, a HitAuditor) error
		want  HitScope
	}{
		{
			name: "cache with scope",
			ctx:  WithHitScope(context.Background(), scope),
			match: func(ctx context.Context, t *testing.T, a HitAuditor) error {
				_, err := auditedCache(t, a).MatchContext(ctx, MatchQuery{Name: queryName, MinScore: 0.9})
				return err
			},
			want: scope,
		},
		{
			name: "cache without scope",
			ctx:  context.Background(),
			match: func(ctx context.Context, t *testing.T, a HitAuditor) error {
				_, err := auditedCache(t, a).MatchContext(ctx, MatchQuery{Name: queryName, MinScore: 0.9})
				return err
			},
			want: HitScope{},
		},
		{
			name: "through ListProvider",
			ctx:  WithHitScope(context.Background(), scope),
			match: func(ctx context.Context, t *testing.T, a HitAuditor) error {
				p, err := NewListProvider(SourceUN, fetchString("a|Synthetic Trading"), parseLines(SourceUN),
					NewCache(WithHitAuditor(a)))
				require.NoError(t, err)
				require.NoError(t, p.Refresh(context.Background()))
				_, err = p.Match(ctx, MatchQuery{Name: queryName, MinScore: 0.9})
				return err
			},
			want: scope,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := &recordingAuditor{}
			require.NoError(t, tc.match(tc.ctx, t, rec))
			calls, scopes, hits := rec.snapshot()
			require.Equal(t, 1, calls)
			assert.Equal(t, tc.want, scopes[0])
			assert.NotEmpty(t, hits)
		})
	}
}

// TestCache_AuditPayloadNeverCarriesQueryName checks the audit batch, as an adapter would serialise it, holds no
// part of the query name. The control proves the detector sees a name when one is there.
func TestCache_AuditPayloadNeverCarriesQueryName(t *testing.T) {
	t.Parallel()
	leaks := func(s string) bool {
		l := strings.ToLower(s)
		return strings.Contains(l, "synthetik") || strings.Contains(l, "tradinq")
	}
	rec := &recordingAuditor{}
	c := auditedCache(t, rec)
	ctx := WithHitScope(context.Background(), HitScope{TenantID: "t", CorrelationID: "c"})
	_, err := c.MatchContext(ctx, MatchQuery{Name: queryName, Country: "BR", MinScore: 0})
	require.NoError(t, err)
	_, scopes, hits := rec.snapshot()
	require.NotEmpty(t, hits)

	payload, err := json.Marshal(struct {
		Scope HitScope
		Hits  []Hit
	}{scopes[0], hits})
	require.NoError(t, err)
	assert.False(t, leaks(string(payload)), "audit payload leaks the query name: %s", payload)

	control, err := json.Marshal(struct{ Name string }{queryName})
	require.NoError(t, err)
	assert.True(t, leaks(string(control)), "negative control: the detector must see a name that is there")
}

func TestCache_AuditFailureFailsClosed(t *testing.T) {
	t.Parallel()
	boom := errors.New("audit sink unavailable")
	c := auditedCache(t, &recordingAuditor{failure: boom})
	got, err := c.Match(MatchQuery{Name: queryName, MinScore: 0.9})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrHitAudit)
	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got, "no verdict without its audit trail")
	assert.NotContains(t, strings.ToLower(err.Error()), "synthetik")
}

func TestCache_NoAuditWhenNothingScored(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		cache func(t *testing.T, a HitAuditor) *Cache
	}{
		{"no candidate", func(t *testing.T, a HitAuditor) *Cache { return auditedCache(t, a) }},
		{"empty cache", func(_ *testing.T, a HitAuditor) *Cache { return NewCache(WithHitAuditor(a)) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := &recordingAuditor{}
			got, err := tc.cache(t, rec).Match(MatchQuery{Name: "Qwxz Vbnm", MinScore: 0.8})
			require.NoError(t, err)
			assert.Empty(t, got)
			calls, _, _ := rec.snapshot()
			assert.Zero(t, calls)
		})
	}
}

func TestCache_DefaultAuditorIsNoop(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]*Cache{"NewCache()": NewCache(), "zero value": {}, "nil auditor": NewCache(WithHitAuditor(nil))} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, c.Load([]Entry{ent("a", "Synthetic Trading")}))
			got, err := c.Match(MatchQuery{Name: queryName, MinScore: 0.9})
			require.NoError(t, err)
			assert.Equal(t, []string{"a"}, ids(got))
		})
	}
}

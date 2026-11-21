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
	batches []HitBatch
	failure error
}

func (r *recordingAuditor) RecordHits(_ context.Context, b HitBatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, b)
	return r.failure
}

func (r *recordingAuditor) snapshot() []HitBatch {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]HitBatch(nil), r.batches...)
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

// testScope is a synthetic, valid scope.
var testScope = HitScope{TenantID: "00000000-0000-4000-8000-000000000001", CorrelationID: "corr-synthetic-1"}

func scoped() context.Context { return WithHitScope(context.Background(), testScope) }

// auditedCache holds one entry above a 0.9 threshold for queryName («a»), one scored below it («d»), and one never
// scored («z»).
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

func TestCache_AuditsDetailAboveThresholdAndAggregateBelow(t *testing.T) {
	t.Parallel()
	rec := &recordingAuditor{}
	c := auditedCache(t, rec)

	got, err := c.MatchContext(scoped(), MatchQuery{Name: queryName, MinScore: 0.9})
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, ids(got), "the threshold still filters what Match returns")

	batches := rec.snapshot()
	require.Len(t, batches, 1, "one batch per Match")
	b := batches[0]
	assert.Equal(t, "SANCTIONS_HIT", HitEventType)
	assert.Equal(t, testScope, b.Scope)
	assert.Equal(t, 0.9, b.MinScore)

	require.Equal(t, []string{"a"}, hitIDs(b.Hits), "only hits at or above the threshold are detailed")
	assert.Equal(t, SourceOFAC, b.Hits[0].Source)
	assert.InDelta(t, got[0].Score, b.Hits[0].Score, 1e-12, "the audited score is the returned score")

	assert.Equal(t, 1, b.BelowThreshold.Count, "«d» is counted; «z» never scored")
	assert.Greater(t, b.BelowThreshold.BestScore, 0.0)
	assert.Less(t, b.BelowThreshold.BestScore, 0.9)
}

// TestCache_NoSubThresholdEntryIDReachesAuditor: below the threshold, the (entry id, score) vector is a
// quasi-identifier of the query, so only the count and the best score reach the auditor.
func TestCache_NoSubThresholdEntryIDReachesAuditor(t *testing.T) {
	t.Parallel()
	rec := &recordingAuditor{}
	c := NewCache(WithHitAuditor(rec))
	require.NoError(t, c.Load([]Entry{
		ent("above-1", "Synthetic Trading"),
		ent("subthr-1", "Synthetic Banana Republic"),
		ent("subthr-2", "Synthetic Apple Grove"),
		ent("subthr-3", "Synthetic Orange Peel Company"),
	}))
	got, err := c.MatchContext(scoped(), MatchQuery{Name: queryName, MinScore: 0.9})
	require.NoError(t, err)
	require.Equal(t, []string{"above-1"}, ids(got))

	batches := rec.snapshot()
	require.Len(t, batches, 1)
	b := batches[0]
	assert.Equal(t, []string{"above-1"}, hitIDs(b.Hits))
	assert.Equal(t, 3, b.BelowThreshold.Count)
	assert.Less(t, b.BelowThreshold.BestScore, 0.9)

	payload, err := json.Marshal(b)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "subthr-", "a sub-threshold entry id reached the auditor: %s", payload)
	assert.Contains(t, string(payload), "above-1")

	control, err := json.Marshal(struct{ IDs []string }{[]string{"subthr-1"}})
	require.NoError(t, err)
	assert.Contains(t, string(control), "subthr-", "negative control: the detector must see an id that is there")
}

func TestCache_AuditCarriesScope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		match func(ctx context.Context, t *testing.T, a HitAuditor) error
	}{
		{
			name: "cache",
			match: func(ctx context.Context, t *testing.T, a HitAuditor) error {
				_, err := auditedCache(t, a).MatchContext(ctx, MatchQuery{Name: queryName, MinScore: 0.9})
				return err
			},
		},
		{
			name: "through ListProvider",
			match: func(ctx context.Context, t *testing.T, a HitAuditor) error {
				p, err := NewListProvider(SourceUN, fetchString("a|Synthetic Trading"), parseLines(SourceUN),
					NewCache(WithHitAuditor(a)))
				require.NoError(t, err)
				require.NoError(t, p.Refresh(context.Background()))
				_, err = p.Match(ctx, MatchQuery{Name: queryName, MinScore: 0.9})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := &recordingAuditor{}
			require.NoError(t, tc.match(scoped(), t, rec))
			batches := rec.snapshot()
			require.Len(t, batches, 1)
			assert.Equal(t, testScope, batches[0].Scope)
			assert.NotEmpty(t, batches[0].Hits)
		})
	}
}

// With an auditor wired, a Match without a tenant fails closed instead of producing a row with no tenant.
func TestCache_AuditRefusesEmptyTenant(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ctx  context.Context
	}{
		{"no scope", context.Background()},
		{"correlation only", WithHitScope(context.Background(), HitScope{CorrelationID: "corr-synthetic-2"})},
		{"blank tenant", WithHitScope(context.Background(), HitScope{TenantID: "   ", CorrelationID: "c"})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := &recordingAuditor{}
			got, err := auditedCache(t, rec).MatchContext(tc.ctx, MatchQuery{Name: queryName, MinScore: 0.9})
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrHitAudit)
			assert.Nil(t, got, "no verdict without a tenant to audit it under")
			assert.Empty(t, rec.snapshot(), "nothing reaches the auditor without a tenant")
			assert.NotContains(t, strings.ToLower(err.Error()), "synthetik")
		})
	}

	t.Run("Match without context", func(t *testing.T) {
		t.Parallel()
		_, err := auditedCache(t, &recordingAuditor{}).Match(MatchQuery{Name: queryName, MinScore: 0.9})
		assert.ErrorIs(t, err, ErrHitAudit, "Match(q) has no scope: not a production path with an auditor")
	})

	t.Run("valid tenant", func(t *testing.T) {
		t.Parallel()
		rec := &recordingAuditor{}
		got, err := auditedCache(t, rec).MatchContext(scoped(), MatchQuery{Name: queryName, MinScore: 0.9})
		require.NoError(t, err)
		assert.Equal(t, []string{"a"}, ids(got))
		require.Len(t, rec.snapshot(), 1)
	})
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
	_, err := c.MatchContext(scoped(), MatchQuery{Name: queryName, Country: "BR", MinScore: 0})
	require.NoError(t, err)
	batches := rec.snapshot()
	require.Len(t, batches, 1)
	require.NotEmpty(t, batches[0].Hits)

	payload, err := json.Marshal(batches[0])
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
	got, err := c.MatchContext(scoped(), MatchQuery{Name: queryName, MinScore: 0.9})
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
			got, err := tc.cache(t, rec).MatchContext(scoped(), MatchQuery{Name: "Qwxz Vbnm", MinScore: 0.8})
			require.NoError(t, err)
			assert.Empty(t, got)
			assert.Empty(t, rec.snapshot())
		})
	}
}

// Without an auditor there is nothing to audit and nothing to refuse: Match works unscoped, as in unit tests.
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

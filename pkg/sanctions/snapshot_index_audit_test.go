package sanctions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// auditedSnapshotIndex is the production shape of a screening pod's index: a fresh snapshot loaded, an auditor wired.
func auditedSnapshotIndex(t *testing.T, clk *testClock, a HitAuditor) *SnapshotIndex {
	t.Helper()
	idx := NewSnapshotIndex(SourceOFAC, MaxSnapshotAge, clk.now, WithHitAuditor(a))
	require.NoError(t, idx.Load(freshSnap(snapTime)))
	return idx
}

func TestSnapshotIndex_IsAContextMatcher(t *testing.T) {
	t.Parallel()
	var x any = NewSnapshotIndex(SourceOFAC, MaxSnapshotAge, nil)
	_, ok := x.(contextMatcher)
	assert.True(t, ok, "ListProvider-style callers must reach MatchContext, so the scope reaches the audit")
}

func TestSnapshotIndex_ScopeReachesAuditor(t *testing.T) {
	t.Parallel()
	a := &recordingAuditor{}
	idx := auditedSnapshotIndex(t, &testClock{}, a)

	hits, err := idx.MatchContext(scoped(), shipQuery())
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	assert.Equal(t, "990002", hits[0].EntryID)

	batches := a.snapshot()
	require.Len(t, batches, 1)
	assert.Equal(t, testScope, batches[0].Scope)
	assert.Contains(t, hitIDs(batches[0].Hits), "990002")
}

func TestSnapshotIndex_AuditRefusesEmptyTenant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		match func(*SnapshotIndex) ([]Match, error)
	}{
		{"Match without context", func(x *SnapshotIndex) ([]Match, error) { return x.Match(shipQuery()) }},
		{"context without scope", func(x *SnapshotIndex) ([]Match, error) {
			return x.MatchContext(context.Background(), shipQuery())
		}},
		{"blank tenant", func(x *SnapshotIndex) ([]Match, error) {
			ctx := WithHitScope(context.Background(), HitScope{TenantID: "  ", CorrelationID: "corr-synthetic-2"})
			return x.MatchContext(ctx, shipQuery())
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := &recordingAuditor{}
			idx := auditedSnapshotIndex(t, &testClock{}, a)
			hits, err := tt.match(idx)
			require.ErrorIs(t, err, ErrHitAudit)
			assert.Nil(t, hits)
			assert.Empty(t, a.snapshot(), "nothing is audited without a tenant")
		})
	}
}

func TestSnapshotIndex_AuditFailureFailsClosed(t *testing.T) {
	t.Parallel()
	a := &recordingAuditor{failure: errors.New("audit store down")}
	idx := auditedSnapshotIndex(t, &testClock{}, a)
	hits, err := idx.MatchContext(scoped(), shipQuery())
	require.ErrorIs(t, err, ErrHitAudit)
	assert.Nil(t, hits)
}

// Staleness is checked first: a stale list blocks with ErrStaleSnapshot whatever the scope, and nothing is audited.
func TestSnapshotIndex_StaleWinsOverAudit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"scoped", scoped()},
		{"no tenant", context.Background()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clk := &testClock{}
			a := &recordingAuditor{}
			idx := auditedSnapshotIndex(t, clk, a)
			clk.set(MaxSnapshotAge + time.Minute)

			hits, err := idx.MatchContext(tt.ctx, shipQuery())
			require.ErrorIs(t, err, ErrStaleSnapshot)
			assert.False(t, errors.Is(err, ErrHitAudit))
			assert.Nil(t, hits)
			assert.Empty(t, a.snapshot())
		})
	}

	t.Run("never loaded", func(t *testing.T) {
		t.Parallel()
		a := &recordingAuditor{}
		idx := NewSnapshotIndex(SourceOFAC, MaxSnapshotAge, (&testClock{}).now, WithHitAuditor(a))
		_, err := idx.MatchContext(context.Background(), shipQuery())
		require.ErrorIs(t, err, ErrStaleSnapshot)
		assert.Empty(t, a.snapshot())
	})
}

// Without an auditor (the no-op default) unscoped Match keeps working.
func TestSnapshotIndex_NoAuditorUnscopedMatch(t *testing.T) {
	t.Parallel()
	idx := NewSnapshotIndex(SourceOFAC, MaxSnapshotAge, (&testClock{}).now)
	require.NoError(t, idx.Load(freshSnap(snapTime)))

	hits, err := idx.Match(shipQuery())
	require.NoError(t, err)
	require.NotEmpty(t, hits)

	hits, err = idx.MatchContext(context.Background(), shipQuery())
	require.NoError(t, err)
	require.NotEmpty(t, hits)
}

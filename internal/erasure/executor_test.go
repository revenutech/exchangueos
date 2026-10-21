package erasure_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/internal/erasure"
)

// ─── fakes ─────────────────────────────────────────────────────────────────

type fakeDB struct {
	begun    int
	commit   int
	rollback int
	execSQLs []string
	execErr  error
	mu       sync.Mutex
}

func (f *fakeDB) BeginTx(_ context.Context) (erasure.Tx, error) {
	f.mu.Lock()
	f.begun++
	f.mu.Unlock()
	return &fakeTx{parent: f}, nil
}

type fakeTx struct{ parent *fakeDB }

func (t *fakeTx) Exec(_ context.Context, sql string, _ ...any) (int64, error) {
	t.parent.mu.Lock()
	t.parent.execSQLs = append(t.parent.execSQLs, sql)
	t.parent.mu.Unlock()
	if t.parent.execErr != nil {
		return 0, t.parent.execErr
	}
	return 3, nil // pretend 3 rows affected per op
}
func (t *fakeTx) Commit(_ context.Context) error {
	t.parent.mu.Lock()
	t.parent.commit++
	t.parent.mu.Unlock()
	return nil
}
func (t *fakeTx) Rollback(_ context.Context) error {
	t.parent.mu.Lock()
	t.parent.rollback++
	t.parent.mu.Unlock()
	return nil
}

type fakeAudit struct {
	ops         []erasure.OpAudit
	completions []erasure.CompletionAudit
}

func (a *fakeAudit) EmitOp(_ context.Context, e erasure.OpAudit) error {
	a.ops = append(a.ops, e)
	return nil
}
func (a *fakeAudit) EmitCompletion(_ context.Context, e erasure.CompletionAudit) error {
	a.completions = append(a.completions, e)
	return nil
}

// ─── BuildSQL ──────────────────────────────────────────────────────────────

func TestBuildSQL_Redact(t *testing.T) {
	plan := &erasure.Plan{
		Ticket: "LGPD-2026-0001",
		Approvals: []string{"dpo", "compliance_officer"},
	}
	sql := erasure.BuildSQL(plan, erasure.Operation{
		Table:  "actors",
		Where:  "id = 'x'",
		Op:     erasure.OpRedact,
		Fields: []string{"name", "email"},
	})
	// Fields sort alphabetically for determinism.
	assert.Equal(t,
		"UPDATE actors SET email = '[REDACTED PER LGPD ART 18 IV LGPD-2026-0001]', name = '[REDACTED PER LGPD ART 18 IV LGPD-2026-0001]' WHERE id = 'x'",
		sql,
	)
}

func TestBuildSQL_HardDelete(t *testing.T) {
	plan := &erasure.Plan{Ticket: "LGPD-2026-0001"}
	sql := erasure.BuildSQL(plan, erasure.Operation{
		Table: "quote_streams",
		Where: "requester_id = 'x'",
		Op:    erasure.OpHardDelete,
	})
	assert.Equal(t, "DELETE FROM quote_streams WHERE requester_id = 'x'", sql)
}

// ─── Executor.Apply ────────────────────────────────────────────────────────

func TestExecutor_Apply_HappyPath(t *testing.T) {
	plan := mustPlan(t, validPlan)
	db := &fakeDB{}
	audit := &fakeAudit{}
	exec := erasure.NewExecutor(db, audit)

	res, err := exec.Apply(context.Background(), plan)
	require.NoError(t, err)

	assert.Equal(t, 2, db.begun, "two operations → two transactions")
	assert.Equal(t, 2, db.commit)
	assert.Equal(t, 0, db.rollback)
	assert.Len(t, db.execSQLs, 2)

	assert.Len(t, audit.ops, 2)
	assert.Len(t, audit.completions, 1)
	assert.Equal(t, int64(6), res.RowsTotal) // 3 rows × 2 ops in the fake

	// Audit metadata propagated correctly.
	assert.Equal(t, "LGPD-2026-0001", audit.ops[0].Ticket)
	assert.Equal(t, "actors", audit.ops[0].Table)
	assert.Equal(t, erasure.OpRedact, audit.ops[0].Op)
	assert.Equal(t, int64(3), audit.ops[0].RowsAffected)
}

func TestExecutor_Apply_RefusesWithoutApprovals(t *testing.T) {
	plan := mustPlan(t, validPlan)
	plan.Approvals = []string{"dpo"} // strip compliance_officer
	exec := erasure.NewExecutor(&fakeDB{}, &fakeAudit{})

	_, err := exec.Apply(context.Background(), plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approvals")
}

func TestExecutor_Apply_RollsBackOnExecError(t *testing.T) {
	plan := mustPlan(t, validPlan)
	db := &fakeDB{execErr: errors.New("simulated CRDB outage")}
	audit := &fakeAudit{}
	exec := erasure.NewExecutor(db, audit)

	_, err := exec.Apply(context.Background(), plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated CRDB outage")

	assert.Equal(t, 1, db.begun, "stopped after first failure")
	assert.Equal(t, 1, db.rollback)
	assert.Equal(t, 0, db.commit)
	assert.Len(t, audit.ops, 0, "no op audit emitted when tx fails")
	assert.Len(t, audit.completions, 0, "no completion when partial apply")
}

func TestExecutor_HashSamples_Stable(t *testing.T) {
	a := erasure.HashSamples([][]string{{"alice", "alice@x"}, {"bob", "bob@x"}})
	b := erasure.HashSamples([][]string{{"alice", "alice@x"}, {"bob", "bob@x"}})
	c := erasure.HashSamples([][]string{{"alice", "alice@x"}, {"carol", "carol@x"}})
	assert.Equal(t, a, b, "same input → same hash")
	assert.NotEqual(t, a, c, "different input → different hash")
	assert.Len(t, a, 64, "sha256 hex = 64 chars")
}

// ─── helpers ───────────────────────────────────────────────────────────────

func mustPlan(t *testing.T, raw string) *erasure.Plan {
	t.Helper()
	p, err := erasure.ParsePlan([]byte(raw))
	require.NoError(t, err)
	return p
}

// Re-export for cross-file test access (validPlan lives in plan_test.go).
var _ = time.Now

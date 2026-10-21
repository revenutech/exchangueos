//go:build integration

// Integration test for cmd/erasure-worker — exercises the Executor against a
// real CockroachDB via the shared dbtest harness. Skips when
// EXCHANGEOS_TEST_DSN is not set.
//
// Coverage: 3 redact ops + 1 hard_delete on seeded rows, completion audit row
// + outbox row landing transactionally, refusal when approvals are stripped
// from the plan.
package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/internal/dbtest"
	"github.com/revenutech/exchangeos/internal/erasure"
)

func TestErasureExecutor_Apply_RedactAndDelete(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	operator := dbtest.SeedTenant(t, pool) // separate "platform" tenant for audit/outbox rows

	// Seed: 3 actors (PII to redact) + 1 counterparty (to hard_delete).
	type actor struct {
		id   uuid.UUID
		sub  string
		name string
	}
	actors := make([]actor, 3)
	for i := range actors {
		a := actor{
			id:   uuid.New(),
			sub:  fmt.Sprintf("sub-%d-%s", i, uuid.New().String()[:8]),
			name: fmt.Sprintf("Subject %d Real Name", i),
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO actors (actor_id, tenant_id, external_sub, type, display_name)
			VALUES ($1, $2, $3, 'HUMAN', $4)`,
			a.id, tenant, a.sub, a.name)
		require.NoError(t, err)
		actors[i] = a
	}
	cpID := dbtest.SeedCounterparty(t, pool, tenant)

	plan, err := erasure.ParsePlan([]byte(fmt.Sprintf(`{
		"ticket": "LGPD-2026-1001",
		"subject_ref": "%s",
		"approvals": ["dpo","compliance_officer"],
		"operations": [
			{"table":"actors","where":"actor_id = '%s'","op":"redact","fields":["display_name","external_sub"]},
			{"table":"actors","where":"actor_id = '%s'","op":"redact","fields":["display_name","external_sub"]},
			{"table":"actors","where":"actor_id = '%s'","op":"redact","fields":["display_name","external_sub"]},
			{"table":"counterparties","where":"counterparty_id = '%s'","op":"hard_delete"}
		]
	}`, uuid.New(), actors[0].id, actors[1].id, actors[2].id, cpID)))
	require.NoError(t, err)

	exec := erasure.NewExecutor(
		erasure.NewPgxDB(pool),
		erasure.NewPgxAudit(pool, operator),
	)
	res, err := exec.Apply(ctx, plan)
	require.NoError(t, err)
	assert.Len(t, res.Ops, 4)
	assert.Equal(t, int64(4), res.RowsTotal, "3 redacts + 1 delete = 4 rows touched")

	// Assert redacted display_names match the marker.
	want := plan.RedactionMarker()
	for _, a := range actors {
		var got string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT display_name FROM actors WHERE actor_id = $1`, a.id,
		).Scan(&got))
		assert.Equal(t, want, got, "actor %s display_name must be redacted", a.id)
	}

	// Assert counterparty gone.
	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM counterparties WHERE counterparty_id = $1)`, cpID,
	).Scan(&exists))
	assert.False(t, exists, "counterparty hard_delete should remove the row")

	// Assert audit_events rows: 4 ops + 1 completion.
	var opAuditCount, doneAuditCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		 WHERE correlation_id = $1 AND event_type = 'LGPD_ERASURE_OP'`,
		plan.Ticket).Scan(&opAuditCount))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		 WHERE correlation_id = $1 AND event_type = 'LGPD_ERASURE_COMPLETED'`,
		plan.Ticket).Scan(&doneAuditCount))
	assert.Equal(t, 4, opAuditCount, "one audit row per op")
	assert.Equal(t, 1, doneAuditCount, "one completion audit row")

	// Assert outbox row: lgpd.erasure_completed.v1.
	var outboxCount int
	var topic, eventName string
	var payload []byte
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM outbox_events
		WHERE topic = 'exchangeos.lgpd.erasure.events.v1'
		  AND event_name = 'lgpd.erasure_completed.v1'
		  AND partition_key = $1`,
		plan.Ticket).Scan(&outboxCount))
	assert.Equal(t, 1, outboxCount, "exactly one outbox event")

	require.NoError(t, pool.QueryRow(ctx, `
		SELECT topic, event_name, event_payload
		FROM outbox_events WHERE partition_key = $1`,
		plan.Ticket).Scan(&topic, &eventName, &payload))
	assert.Equal(t, "exchangeos.lgpd.erasure.events.v1", topic)
	assert.Equal(t, "lgpd.erasure_completed.v1", eventName)

	var decoded struct {
		Ticket      string `json:"ticket"`
		OpsExecuted int    `json:"ops_executed"`
		RowsTotal   int64  `json:"rows_total"`
	}
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, plan.Ticket, decoded.Ticket)
	assert.Equal(t, 4, decoded.OpsExecuted)
	assert.Equal(t, int64(4), decoded.RowsTotal)
}

func TestErasureExecutor_RefusesWithoutApprovals(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	operator := dbtest.SeedTenant(t, pool)

	cpID := dbtest.SeedCounterparty(t, pool, tenant)

	// Bypass ParsePlan (which would enforce approvals via schema) — construct
	// directly to simulate a tampered/stale plan reaching the executor.
	plan := &erasure.Plan{
		Ticket:     "LGPD-2026-1002",
		SubjectRef: uuid.New().String(),
		Approvals:  []string{"dpo"}, // missing compliance_officer
		Operations: []erasure.Operation{
			{Table: "counterparties", Where: fmt.Sprintf("counterparty_id = '%s'", cpID), Op: erasure.OpHardDelete},
		},
	}

	exec := erasure.NewExecutor(
		erasure.NewPgxDB(pool),
		erasure.NewPgxAudit(pool, operator),
	)
	_, err := exec.Apply(ctx, plan)
	require.Error(t, err)

	// Counterparty must still exist — no mutation should have happened.
	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM counterparties WHERE counterparty_id = $1)`, cpID,
	).Scan(&exists))
	assert.True(t, exists, "missing-approvals plan must not delete anything")

	// No audit rows for this ticket either.
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events WHERE correlation_id = $1`,
		plan.Ticket).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestErasureExecutor_RollsBackOnSQLError(t *testing.T) {
	pool := dbtest.PoolOrSkip(t)
	ctx := context.Background()
	tenant := dbtest.SeedTenant(t, pool)
	operator := dbtest.SeedTenant(t, pool)

	cpID := dbtest.SeedCounterparty(t, pool, tenant)

	// Second op targets a non-existent table → tx fails, no audit emitted for
	// the failed op, but the first op already committed (per executor design).
	plan := &erasure.Plan{
		Ticket:     "LGPD-2026-1003",
		SubjectRef: uuid.New().String(),
		Approvals:  []string{"dpo", "compliance_officer"},
		Operations: []erasure.Operation{
			{Table: "counterparties", Where: fmt.Sprintf("counterparty_id = '%s'", cpID), Op: erasure.OpHardDelete},
			{Table: "this_table_does_not_exist", Where: "1=1", Op: erasure.OpHardDelete},
		},
	}

	exec := erasure.NewExecutor(
		erasure.NewPgxDB(pool),
		erasure.NewPgxAudit(pool, operator),
	)
	res, err := exec.Apply(ctx, plan)
	require.Error(t, err)
	// First op succeeded + audit committed.
	assert.Len(t, res.Ops, 1)

	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		 WHERE correlation_id = $1 AND event_type = 'LGPD_ERASURE_OP'`,
		plan.Ticket).Scan(&n))
	assert.Equal(t, 1, n, "successful op leaves audit row; failed op does not")

	// No completion audit (executor returns before emitting it).
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		 WHERE correlation_id = $1 AND event_type = 'LGPD_ERASURE_COMPLETED'`,
		plan.Ticket).Scan(&n))
	assert.Equal(t, 0, n)
}

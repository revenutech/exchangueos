package erasure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// DB is the minimal driver-agnostic DB surface the Executor needs.
// Both pgx/v5 pools and database/sql connections satisfy this with thin adapters,
// keeping the package free of vendor imports until integration time.
type DB interface {
	BeginTx(ctx context.Context) (Tx, error)
}

type Tx interface {
	Exec(ctx context.Context, sql string, args ...any) (rowsAffected int64, err error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// AuditEmitter writes one row per executed op + a final completion event.
// In production this wraps modules/admin's audit pipeline + the outbox.
type AuditEmitter interface {
	EmitOp(ctx context.Context, evt OpAudit) error
	EmitCompletion(ctx context.Context, evt CompletionAudit) error
}

// OpAudit is one audit_event payload emitted per Operation.
type OpAudit struct {
	Ticket       string    `json:"ticket"`
	SubjectRef   string    `json:"subject_ref"`
	Table        string    `json:"table"`
	Op           Op        `json:"op"`
	RowsAffected int64     `json:"rows_affected"`
	BeforeHash   string    `json:"before_hash"`
	AfterHash    string    `json:"after_hash"`
	AppliedAt    time.Time `json:"applied_at"`
}

// CompletionAudit is the single rollup event emitted after the final commit;
// feeds the outbox topic `lgpd.erasure_completed.v1`.
type CompletionAudit struct {
	Ticket      string    `json:"ticket"`
	SubjectRef  string    `json:"subject_ref"`
	OpsExecuted int       `json:"ops_executed"`
	RowsTotal   int64     `json:"rows_total"`
	CompletedAt time.Time `json:"completed_at"`
}

// Result aggregates per-op outcomes for the caller (typically cmd/erasure-worker).
type Result struct {
	Ops       []OpAudit
	RowsTotal int64
}

// Executor applies a validated Plan against a DB, emitting audit events.
//
// Concurrency: not safe for concurrent use; one executor per ticket.
type Executor struct {
	db    DB
	audit AuditEmitter
	clock func() time.Time // injectable for tests
}

// NewExecutor wires dependencies. The clock defaults to time.Now() but can be
// overridden in tests for deterministic audit timestamps.
func NewExecutor(db DB, audit AuditEmitter) *Executor {
	return &Executor{db: db, audit: audit, clock: func() time.Time { return time.Now().UTC() }}
}

// Apply runs the plan inside a single transaction per operation. Order is preserved.
// On any operation failure, the transaction rolls back and the partially-emitted
// audit events are NOT undone — they remain as evidence that a partial attempt
// happened (per ISO 27001 control 8.15 logging).
func (e *Executor) Apply(ctx context.Context, plan *Plan) (*Result, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if !plan.HasRequiredApprovals() {
		return nil, fmt.Errorf("erasure: refusing to apply: plan lacks dpo+compliance_officer approvals")
	}

	result := &Result{Ops: make([]OpAudit, 0, len(plan.Operations))}

	for i, op := range plan.Operations {
		audit, err := e.applyOne(ctx, plan, op)
		if err != nil {
			return result, fmt.Errorf("op[%d] %s: %w", i, op.Table, err)
		}
		result.Ops = append(result.Ops, audit)
		result.RowsTotal += audit.RowsAffected
	}

	completion := CompletionAudit{
		Ticket:      plan.Ticket,
		SubjectRef:  plan.SubjectRef,
		OpsExecuted: len(result.Ops),
		RowsTotal:   result.RowsTotal,
		CompletedAt: e.clock(),
	}
	if err := e.audit.EmitCompletion(ctx, completion); err != nil {
		return result, fmt.Errorf("emit completion: %w", err)
	}
	return result, nil
}

// applyOne runs a single Operation inside its own transaction.
func (e *Executor) applyOne(ctx context.Context, plan *Plan, op Operation) (OpAudit, error) {
	sql := BuildSQL(plan, op)

	tx, err := e.db.BeginTx(ctx)
	if err != nil {
		return OpAudit{}, fmt.Errorf("begin: %w", err)
	}

	rows, err := tx.Exec(ctx, sql)
	if err != nil {
		_ = tx.Rollback(ctx)
		return OpAudit{}, fmt.Errorf("exec: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OpAudit{}, fmt.Errorf("commit: %w", err)
	}

	audit := OpAudit{
		Ticket:       plan.Ticket,
		SubjectRef:   plan.SubjectRef,
		Table:        op.Table,
		Op:           op.Op,
		RowsAffected: rows,
		BeforeHash:   "n/a", // hash diff is MS-024a Stage 3 (snapshot pre/post)
		AfterHash:    "n/a",
		AppliedAt:    e.clock(),
	}
	if err := e.audit.EmitOp(ctx, audit); err != nil {
		// Audit failure is not fatal — the data mutation already committed —
		// but caller MUST handle it (typically: page on-call + manual reconcile).
		return audit, fmt.Errorf("emit op audit: %w", err)
	}
	return audit, nil
}

// BuildSQL returns the exact SQL string this executor would issue for an op.
// Exported so cmd/erasure-worker --dry-run can render it without running it.
//
// SECURITY NOTE: the WHERE clause is interpolated verbatim from the plan, NOT
// parameterised. This is intentional — the plan is co-signed by DPO+Compliance
// and reviewed during Stage 3 of the workflow; the caller is trusted. We refuse
// to apply any plan with an empty WHERE (Plan.validate).
func BuildSQL(plan *Plan, op Operation) string {
	switch op.Op {
	case OpRedact:
		return buildRedactSQL(plan, op)
	case OpHardDelete:
		return fmt.Sprintf("DELETE FROM %s WHERE %s", op.Table, op.Where)
	default:
		return "" // unreachable: Plan.validate rejects unknown ops
	}
}

func buildRedactSQL(plan *Plan, op Operation) string {
	marker := plan.RedactionMarker()
	// Each value is the marker plus a per-row random token: a redacted field may sit under a UNIQUE constraint
	// (actors has UNIQUE (tenant_id, external_sub)), and one shared marker would refuse the second row of a ticket.
	// The token is random, not derived from the old value: a hash of the PII would still link the row to the subject.
	// Deterministic field order for reproducible SQL across runs.
	fields := append([]string(nil), op.Fields...)
	sort.Strings(fields)
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprintf("%s = '%s ' || gen_random_uuid()::STRING", f, escapeSingleQuote(marker))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s", op.Table, strings.Join(parts, ", "), op.Where)
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// HashSamples is a tiny helper for the audit before/after hash columns once the
// snapshot pipeline lands in Stage 3. Currently unused — kept here so the
// upgrade path is obvious to future contributors.
func HashSamples(rows [][]string) string {
	h := sha256.New()
	for _, r := range rows {
		for _, c := range r {
			h.Write([]byte(c))
			h.Write([]byte{0}) // separator
		}
		h.Write([]byte{1}) // row separator
	}
	return hex.EncodeToString(h.Sum(nil))
}

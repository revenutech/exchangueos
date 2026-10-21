// cmd/erasure-worker — LGPD Art. 18 IV right-to-erasure executor.
//
// Usage:
//   erasure-worker --ticket LGPD-2026-0001 --plan plan.yaml --dry-run
//   erasure-worker --ticket LGPD-2026-0001 --plan plan.yaml --execute
//
// --dry-run prints the SQL each operation would issue + the affected row count
// per the WHERE clause, WITHOUT mutating data. Safe to run any time.
//
// --execute requires the plan to carry approvals from both 'dpo' AND
// 'compliance_officer' roles (4-eyes per workflow Stage 3) AND env var
// EXCHANGEOS_ERASURE_CONFIRM=YES-I-MEAN-IT (a final out-of-band guard against
// accidental invocation in CI / shell history).
//
// Successful execution emits one audit_event per operation and one outbox event
// `lgpd.erasure_completed.v1` after the final commit so downstream modules
// (LedgerOS, ComplOS) can apply equivalent erasure to their own datasets.
//
// Full reference: docs/security/data-lifecycle/erasure-workflow.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/internal/erasure"
)

// Required approver roles for --execute. Matches the AC for MS-024a:
//   EXCHANGEOS_ERASURE_APPROVERS=dpo,compliance_officer
const (
	roleDPO        = "dpo"
	roleCompliance = "compliance_officer"
)

func main() {
	if err := run(); err != nil {
		slog.Error("erasure-worker: failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		ticket   = flag.String("ticket", "", "LGPD ticket ID (must match plan.ticket)")
		planPath = flag.String("plan", "", "path to signed plan YAML")
		dryRun   = flag.Bool("dry-run", false, "print operations + row counts without mutating")
		execute  = flag.Bool("execute", false, "actually apply the plan (requires 4-eyes approval + confirm env var)")
	)
	flag.Parse()

	if *ticket == "" || *planPath == "" {
		return fmt.Errorf("--ticket and --plan are required")
	}
	if *dryRun == *execute {
		return fmt.Errorf("exactly one of --dry-run or --execute must be set")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	data, err := os.ReadFile(*planPath)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}
	plan, err := erasure.ParsePlan(data)
	if err != nil {
		return fmt.Errorf("parse plan: %w", err)
	}
	if plan.Ticket != *ticket {
		return fmt.Errorf("ticket mismatch: --ticket=%s vs plan.ticket=%s", *ticket, plan.Ticket)
	}

	slog.Info("erasure-worker: plan loaded",
		"ticket", plan.Ticket,
		"subject_ref", plan.SubjectRef,
		"operations", len(plan.Operations),
		"approvals", plan.Approvals,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if *dryRun {
		return dryRunPlan(ctx, plan)
	}

	// ── execute branch ──────────────────────────────────────────────────────
	if !plan.HasRequiredApprovals() {
		return fmt.Errorf("plan lacks required approvals (need both 'dpo' AND 'compliance_officer')")
	}
	if err := checkApproverEnv(); err != nil {
		return err
	}
	if os.Getenv("EXCHANGEOS_ERASURE_CONFIRM") != "YES-I-MEAN-IT" {
		return fmt.Errorf("EXCHANGEOS_ERASURE_CONFIRM=YES-I-MEAN-IT required for --execute")
	}

	dsn := os.Getenv("EXCHANGEOS_DB_DSN")
	if dsn == "" {
		return fmt.Errorf("EXCHANGEOS_DB_DSN required for --execute")
	}
	operatorTenant, err := operatorTenantFromEnv()
	if err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	exec := erasure.NewExecutor(
		erasure.NewPgxDB(pool),
		erasure.NewPgxAudit(pool, operatorTenant),
	)
	res, err := exec.Apply(ctx, plan)
	if err != nil {
		slog.Error("erasure-worker: apply failed",
			"ticket", plan.Ticket, "ops_done", len(res.Ops), "rows_done", res.RowsTotal, "err", err)
		return err
	}
	slog.Info("erasure-worker: applied",
		"ticket", plan.Ticket, "ops", len(res.Ops), "rows", res.RowsTotal)
	return nil
}

// checkApproverEnv parses EXCHANGEOS_ERASURE_APPROVERS as a comma-separated
// list and refuses execute unless both required roles are present. This is a
// belt-and-braces check on top of plan.HasRequiredApprovals — operators can
// deliberately withhold the env var to block --execute even when a fully-
// approved plan is on disk.
func checkApproverEnv() error {
	raw := os.Getenv("EXCHANGEOS_ERASURE_APPROVERS")
	if raw == "" {
		return fmt.Errorf("EXCHANGEOS_ERASURE_APPROVERS env var required for --execute (e.g. 'dpo,compliance_officer')")
	}
	have := map[string]bool{}
	for _, a := range strings.Split(raw, ",") {
		have[strings.ToLower(strings.TrimSpace(a))] = true
	}
	if !have[roleDPO] || !have[roleCompliance] {
		return fmt.Errorf("EXCHANGEOS_ERASURE_APPROVERS must contain both %q and %q (got %q)",
			roleDPO, roleCompliance, raw)
	}
	return nil
}

// operatorTenantFromEnv parses EXCHANGEOS_OPERATOR_TENANT_ID. The audit_events
// + outbox_events rows both carry NOT NULL tenant_id; this is the tenant under
// whose name the regulatory action was taken (typically the platform tenant).
func operatorTenantFromEnv() (uuid.UUID, error) {
	raw := os.Getenv("EXCHANGEOS_OPERATOR_TENANT_ID")
	if raw == "" {
		return uuid.Nil, fmt.Errorf("EXCHANGEOS_OPERATOR_TENANT_ID required for --execute")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("EXCHANGEOS_OPERATOR_TENANT_ID parse: %w", err)
	}
	return id, nil
}

// dryRunPlan prints the SQL each op would issue + the row count it would
// affect. Read-only — never mutates. When EXCHANGEOS_DB_DSN is set the DSN is
// used to issue `SELECT count(*) FROM <table> WHERE <where>` per op so the
// operator gets a precise blast-radius preview; without it the SQL is still
// printed (useful for offline plan review).
func dryRunPlan(ctx context.Context, plan *erasure.Plan) error {
	var pool *pgxpool.Pool
	if dsn := os.Getenv("EXCHANGEOS_DB_DSN"); dsn != "" {
		p, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("open db (dry-run): %w", err)
		}
		defer p.Close()
		if err := p.Ping(ctx); err != nil {
			return fmt.Errorf("ping db (dry-run): %w", err)
		}
		pool = p
	}

	fmt.Printf("\n=== DRY RUN — ticket %s ===\n\n", plan.Ticket)
	var totalRows int64
	for i, op := range plan.Operations {
		sql := erasure.BuildSQL(plan, op)
		fmt.Printf("[%d] %s\n", i+1, sql)
		if pool != nil {
			n, err := countAffected(ctx, pool, op)
			if err != nil {
				return fmt.Errorf("op[%d] count: %w", i, err)
			}
			fmt.Printf("    → would affect %d row(s)\n", n)
			totalRows += n
		}
	}
	fmt.Printf("\n=== END DRY RUN (%d ops, %d rows total) ===\n\n",
		len(plan.Operations), totalRows)
	slog.Info("erasure-worker: dry-run complete",
		"operations", len(plan.Operations), "rows_estimated", totalRows)
	return nil
}

// countAffected runs `SELECT count(*) FROM <table> WHERE <where>` against the
// pool — never mutates. Read-only by construction.
func countAffected(ctx context.Context, pool *pgxpool.Pool, op erasure.Operation) (int64, error) {
	q := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", op.Table, op.Where)
	var n int64
	if err := pool.QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

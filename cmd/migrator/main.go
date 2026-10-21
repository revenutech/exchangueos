// Package main — exchangeos-migrator: applies SQL migrations from migrations/ to shared CRDB hub.
//
// Usage:
//
//	exchangeos-migrator up           # apply all pending up migrations
//	exchangeos-migrator up <N>       # apply at most N up migrations
//	exchangeos-migrator down <N>     # roll back N migrations
//	exchangeos-migrator status       # print current version + dirty flag
//	exchangeos-migrator force <ver>  # set version (rescue after manual fix; ALSO clears dirty)
//	exchangeos-migrator seed         # load seeds/ (idempotent)
//
// Driver: golang-migrate/v4 with the pgx-backed cockroachdb driver.
// DSN: EXCHANGEOS_DB_DSN — production MUST use the shared CRDB hub TLS DSN
// (NEVER inline --insecure). The DSN is logged with the password redacted.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/cockroachdb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/internal/config"
	"github.com/revenutech/exchangeos/internal/telemetry"
)

const (
	serviceName    = "exchangeos-migrator"
	serviceVersion = "0.1.0-dev"
	defaultSource  = "file://migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"up"}
	}
	cmd := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger, err := telemetry.NewLogger(cfg.Env)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	if cfg.DB.DSN == "" {
		return errors.New("EXCHANGEOS_DB_DSN is required (use shared CRDB hub TLS DSN in production)")
	}

	source := getEnvOr("EXCHANGEOS_MIGRATIONS_SOURCE", defaultSource)
	logger.Info("migrator starting",
		zap.String("cmd", cmd),
		zap.String("env", cfg.Env),
		zap.String("dsn", cfg.DB.Redacted()),
		zap.String("source", source),
	)

	m, err := migrate.New(source, cockroachDSN(cfg.DB.DSN))
	if err != nil {
		return fmt.Errorf("migrate.New: %w", scrubDSNFromError(err, cfg.DB.DSN))
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			logger.Warn("migrator close", zap.Errors("errors", []error{srcErr, dbErr}))
		}
	}()

	switch cmd {
	case "up":
		return doUp(m, logger, args[1:])
	case "down":
		return doDown(m, logger, args[1:])
	case "status":
		return doStatus(m, logger)
	case "force":
		return doForce(m, logger, args[1:])
	case "seed":
		seedDir := getEnvOr("EXCHANGEOS_SEEDS_DIR", "seeds")
		return doSeed(cfg.DB.DSN, seedDir, logger)
	default:
		return fmt.Errorf("unknown command %q (use: up|down|status|force|seed)", cmd)
	}
}

func doUp(m *migrate.Migrate, logger *zap.Logger, args []string) error {
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			return fmt.Errorf("up <N>: N must be positive int, got %q", args[0])
		}
		err = m.Steps(n)
		return classify(err, logger, fmt.Sprintf("up %d", n))
	}
	return classify(m.Up(), logger, "up")
}

func doDown(m *migrate.Migrate, logger *zap.Logger, args []string) error {
	if len(args) == 0 {
		return errors.New("down <N>: N is required to avoid accidental full rollback")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 {
		return fmt.Errorf("down <N>: N must be positive int, got %q", args[0])
	}
	return classify(m.Steps(-n), logger, fmt.Sprintf("down %d", n))
}

func doStatus(m *migrate.Migrate, logger *zap.Logger) error {
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		logger.Info("status: no migrations applied yet")
		return nil
	}
	if err != nil {
		return fmt.Errorf("version: %w", err)
	}
	logger.Info("status", zap.Uint("version", v), zap.Bool("dirty", dirty))
	if dirty {
		return errors.New("dirty state detected — fix manually then `force <ver>` to clear")
	}
	return nil
}

func doForce(m *migrate.Migrate, logger *zap.Logger, args []string) error {
	if len(args) == 0 {
		return errors.New("force <ver>: ver is required")
	}
	v, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("force <ver>: ver must be int, got %q", args[0])
	}
	if err := m.Force(v); err != nil {
		return fmt.Errorf("force: %w", err)
	}
	logger.Warn("forced version (dirty flag cleared)", zap.Int("version", v))
	return nil
}

// classify treats ErrNoChange / ErrNilVersion as success — they are common no-op outcomes.
func classify(err error, logger *zap.Logger, op string) error {
	switch {
	case err == nil:
		logger.Info("migrate ok", zap.String("op", op))
		return nil
	case errors.Is(err, migrate.ErrNoChange):
		logger.Info("migrate no-change", zap.String("op", op))
		return nil
	case errors.Is(err, migrate.ErrNilVersion):
		logger.Info("migrate nil-version", zap.String("op", op))
		return nil
	default:
		return fmt.Errorf("migrate %s: %w", op, err)
	}
}

// cockroachDSN normalises the DSN to the scheme golang-migrate's cockroachdb driver expects.
// Accepts both "postgres://..." and "cockroach://..." inputs.
func cockroachDSN(dsn string) string {
	const prefix = "postgres://"
	if len(dsn) >= len(prefix) && dsn[:len(prefix)] == prefix {
		return "cockroach://" + dsn[len(prefix):]
	}
	return dsn
}

// scrubDSNFromError guards against a real leak in golang-migrate itself:
// migrate.New wraps database.Open failures with `fmt.Errorf("failed to open
// database, %q: %w", databaseURL, err)` — i.e. it embeds the RAW DSN
// (including the password) into the error message whenever the driver fails
// to open the DB (bad host, refused connection, auth failure — all reachable
// in normal operation). That error bubbles up through `run()` to
// `fmt.Fprintf(os.Stderr, "fatal: %v\n", err)` in main(), printing the
// password in clear text. Verified live against
// github.com/golang-migrate/migrate/v4 v4.18.1 (database/driver.go Open()).
//
// rawDSN is the config's original DSN (cfg.DB.DSN); scrubDSNFromError also
// checks the cockroachDSN(rawDSN) transform since that's the literal string
// passed to migrate.New. Both occurrences are replaced with the value
// config.DBConfig.Redacted() would have produced.
func scrubDSNFromError(err error, rawDSN string) error {
	if err == nil || rawDSN == "" {
		return err
	}
	msg := err.Error()
	transformed := cockroachDSN(rawDSN)
	if !strings.Contains(msg, rawDSN) && !strings.Contains(msg, transformed) {
		return err
	}
	redacted := config.DBConfig{DSN: rawDSN}.Redacted()
	msg = strings.ReplaceAll(msg, rawDSN, redacted)
	msg = strings.ReplaceAll(msg, transformed, redacted)
	return errors.New(msg)
}

// doSeed loads all *.sql files from seedDir in lexicographic order and runs
// each as a single PostgreSQL session (the file's own BEGIN/COMMIT controls tx).
// Seeds MUST be idempotent (ON CONFLICT DO NOTHING) — re-running is a no-op.
func doSeed(dsn, seedDir string, logger *zap.Logger) error {
	abs, err := filepath.Abs(seedDir)
	if err != nil {
		return fmt.Errorf("seed: resolve %q: %w", seedDir, err)
	}
	entries, err := filepath.Glob(filepath.Join(abs, "*.sql"))
	if err != nil {
		return fmt.Errorf("seed: glob: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("seed: no *.sql files in %s", abs)
	}
	sort.Strings(entries)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("seed: connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	for _, path := range entries {
		name := filepath.Base(path)
		buf, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("seed: read %s: %w", name, err)
		}
		// The seed file ships with its own BEGIN/COMMIT; we just submit it.
		if _, err := conn.Exec(ctx, string(buf)); err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}
		logger.Info("seed applied", zap.String("file", name))
	}
	logger.Info("seed complete", zap.Int("files", len(entries)))
	return nil
}

func getEnvOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

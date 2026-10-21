// Package db — pgx/v5 connection pool factory shared by infrastructure repos.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/internal/config"
)

// New constructs a pgx connection pool from config.
//
// Production DSN must include sslmode=verify-full + sslrootcert pointing at the
// shared CRDB hub CA (NEVER --insecure). The pool honours MinConn/MaxConn from config.
func New(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("db: dsn is empty")
	}
	pcfg, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	if cfg.DB.MaxConn > 0 {
		pcfg.MaxConns = int32(cfg.DB.MaxConn)
	}
	if cfg.DB.MinConn > 0 {
		pcfg.MinConns = int32(cfg.DB.MinConn)
	}
	pcfg.MaxConnLifetime = 30 * time.Minute
	pcfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("db: new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

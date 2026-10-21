// Package main — exchangeos-worker: outbox dispatch loop + (future) Kafka consumers.
//
// Dispatches `outbox_events` rows to Kafka in commit order with backoff between
// empty batches. Publisher is pluggable via build tag:
//
//	default       → pkg/outbox/PublisherFunc no-op (logs only)
//	-tags kafka   → pkg/outbox/kafka (franz-go, brokers from env)
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/internal/config"
	"github.com/revenutech/exchangeos/internal/db"
	"github.com/revenutech/exchangeos/internal/telemetry"
	"github.com/revenutech/exchangeos/pkg/outbox"
	outboxpg "github.com/revenutech/exchangeos/pkg/outbox/postgres"
)

const (
	serviceName    = "exchangeos-worker"
	serviceVersion = "0.1.0-dev"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger, err := telemetry.NewLogger(cfg.Env)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("worker starting",
		zap.String("env", cfg.Env),
		zap.String("version", serviceVersion),
		zap.String("repo_backend", cfg.Repos.Backend),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if cfg.Repos.Backend != "postgres" {
		// Memory backend is for tests + bootstrap; worker dispatch is a no-op there.
		logger.Info("worker idle — outbox dispatch requires postgres backend")
		<-ctx.Done()
		logger.Info("worker stopped")
		return nil
	}

	pool, err := db.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()

	store := outboxpg.NewStore(pool)
	pub := newPublisher(logger) // selected via build tag (see publisher_*.go)
	defer closePublisher(pub)

	logger.Info("worker dispatch loop starting",
		zap.String("publisher", publisherName()),
		zap.Int("batch_size", 100),
	)

	return dispatchLoop(ctx, logger, store, pub, dispatchLoopOpts{
		BatchSize: 100,
		EmptyWait: 500 * time.Millisecond,
		ErrorWait: 2 * time.Second,
	})
}

type dispatchLoopOpts struct {
	BatchSize int
	EmptyWait time.Duration
	ErrorWait time.Duration
}

// dispatchLoop is the worker hot path: take a batch, publish, sleep on empty or error.
func dispatchLoop(ctx context.Context, logger *zap.Logger, store outbox.Store, pub outbox.Publisher, opts dispatchLoopOpts) error {
	ticker := time.NewTicker(opts.EmptyWait)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("dispatch loop exit signal received")
			return nil
		default:
		}
		n, err := outbox.Dispatch(ctx, store, pub, opts.BatchSize)
		switch {
		case err != nil:
			logger.Warn("dispatch batch error", zap.Int("dispatched", n), zap.Error(err))
			sleep(ctx, opts.ErrorWait)
		case n == 0:
			sleep(ctx, opts.EmptyWait)
		default:
			logger.Debug("batch dispatched", zap.Int("count", n))
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

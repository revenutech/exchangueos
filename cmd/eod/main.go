// Package main — exchangeos-eod: end-of-day batch (PTAX fixing, MTM, position snapshot, BACEN reports).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/internal/config"
	"github.com/revenutech/exchangeos/internal/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	logger, _ := telemetry.NewLogger(cfg.Env)
	defer func() { _ = logger.Sync() }()

	logger.Info("eod starting", zap.String("env", cfg.Env))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// TODO MS-023e/f2: implement EOD pipeline (PTAX, MTM revaluation, position snapshot, BACEN SISBACEN/CCS/CAMBIO)
	<-ctx.Done()
	logger.Info("eod stopped")
}

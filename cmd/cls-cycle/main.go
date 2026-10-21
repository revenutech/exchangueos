// Package main — exchangeos-cls-cycle: opens/closes daily CLS cycles per CET schedule.
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

	logger.Info("cls-cycle starting", zap.String("env", cfg.Env))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// TODO MS-023d: implement cron loop honoring CLS cycle CET schedule
	//   07:00 OpenCycle, 08:00/09:00/10:00 PayIn deadlines, 12:00 CloseCycle
	<-ctx.Done()
	logger.Info("cls-cycle stopped")
}

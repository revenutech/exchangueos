// Package main — exchangeos-mq-bridge: legacy SWIFT MT ↔ ISO 20022 fxtr bridge (transitional).
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

	logger.Info("mq-bridge starting", zap.String("env", cfg.Env))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// TODO MS-023g: implement SWIFT MT (MT300/MT304/MT320) ↔ fxtr.014/015/016 translator
	<-ctx.Done()
	logger.Info("mq-bridge stopped")
}

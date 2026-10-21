// Package main — exchangeos-cred-rotator: rotates 14 M2M client_secrets via Vault SPI (30d cadence).
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

	logger.Info("cred-rotator starting", zap.String("env", cfg.Env))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// TODO MS-023q: implement rotation loop:
	//   - read M2M client list from KeycloakOS
	//   - for each: generate new secret → push to Vault SPI → patch Keycloak
	//   - cadence: 30 days; emit OTel span per rotation
	<-ctx.Done()
	logger.Info("cred-rotator stopped")
}

// Package main — exchangeos-api: dual HTTP (:8084) + gRPC (:9084) server.
//
// Bootstraps: config → telemetry → DB pool → gRPC server → HTTP server → graceful shutdown.
// Health endpoints: /healthz (liveness) + /readyz (readiness) + gRPC HealthService.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/revenutech/exchangeos/internal/adminapi"
	"github.com/revenutech/exchangeos/internal/config"
	"github.com/revenutech/exchangeos/internal/container"
	"github.com/revenutech/exchangeos/internal/telemetry"
)

const (
	serviceName    = "exchangeos-api"
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

	logger.Info("exchangeos-api starting",
		zap.String("env", cfg.Env),
		zap.String("version", serviceVersion),
		zap.Int("http_port", cfg.HTTP.Port),
		zap.Int("grpc_port", cfg.GRPC.Port),
	)

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	shutdownTelemetry, err := telemetry.Init(rootCtx, telemetry.Options{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		OTLPEndpoint:   cfg.OTel.Endpoint,
		Env:            cfg.Env,
	})
	if err != nil {
		return fmt.Errorf("telemetry init: %w", err)
	}
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = shutdownTelemetry(shutdownCtx)
	}()

	di, err := container.New(rootCtx, cfg)
	if err != nil {
		return fmt.Errorf("container: %w", err)
	}
	defer di.Close()
	logger.Info("application container initialised",
		zap.String("repo_backend", cfg.Repos.Backend),
		zap.String("pricing", "stub (CIP/cross-rate wiring pending — see internal/container)"),
	)

	grpcServer, grpcLn, err := buildGRPC(cfg, logger, di)
	if err != nil {
		return fmt.Errorf("grpc: %w", err)
	}
	httpServer := buildHTTP(cfg, logger, di)

	errCh := make(chan error, 2)
	go func() {
		logger.Info("grpc server listening", zap.String("addr", grpcLn.Addr().String()))
		if err := grpcServer.Serve(grpcLn); err != nil {
			errCh <- fmt.Errorf("grpc serve: %w", err)
		}
	}()
	go func() {
		logger.Info("http server listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http serve: %w", err)
		}
	}()

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", zap.Error(err))
		cancel()
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	logger.Info("shutting down http")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http shutdown error", zap.Error(err))
	}

	logger.Info("shutting down grpc")
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}

	logger.Info("exchangeos-api stopped")
	return nil
}

func buildGRPC(cfg *config.Config, logger *zap.Logger, di *container.Container) (*grpc.Server, net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		return nil, nil, err
	}
	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvBytes),
		grpc.MaxSendMsgSize(cfg.GRPC.MaxSendBytes),
	)
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, hs)
	if cfg.GRPC.Reflection {
		reflection.Register(srv)
	}

	// Bounded-context service registration. Gated by build tag `grpcgen`:
	//   default build (no tag)         → no-op (grpc_register_default.go)
	//   build with -tags grpcgen       → registers all generated services (grpc_register_proto.go)
	// Requires `task proto:gen` to have produced proto/gen/exchangeos/v1/*.pb.go.
	registerGeneratedServices(srv, di)

	logger.Info("grpc services registered",
		zap.Bool("reflection", cfg.GRPC.Reflection),
		zap.String("note", "use -tags grpcgen after `task proto:gen` to bind bounded-context services"),
	)
	// Tracking (registered when grpcgen tag is on):
	// Tracking:
	//   MS-023b — QuoteServiceServer, RefDataServiceServer
	//   MS-023c — TradeServiceServer
	//   MS-023d — SettlementServiceServer
	//   MS-023e — RiskServiceServer, PositionServiceServer
	//   MS-023f — ComplianceServiceServer, AdminServiceServer
	return srv, ln, nil
}

func buildHTTP(cfg *config.Config, logger *zap.Logger, di *container.Container) *http.Server {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName, "version": serviceVersion})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": serviceName, "version": serviceVersion, "env": cfg.Env})
	})

	// ── /v1/refdata/currencies — smoke endpoint proving container wiring ───
	// Replaced by the gRPC-gateway-generated handler once proto/gen lands.
	r.GET("/v1/refdata/currencies", func(c *gin.Context) {
		activeOnly := c.Query("active_only") == "true"
		list, err := di.RefData.ListCurrencies(c.Request.Context(), activeOnly)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]gin.H, 0, len(list))
		for _, cur := range list {
			out = append(out, gin.H{
				"code":            cur.Code(),
				"name":            cur.Name(),
				"minor_units":     cur.MinorUnits(),
				"cls_eligible":    cur.IsCLSEligible(),
				"cfets_eligible":  cur.IsCFETSEligible(),
				"active":          cur.IsActive(),
			})
		}
		c.JSON(http.StatusOK, gin.H{"currencies": out, "count": len(out)})
	})

	// ── /v1/trades/:id — smoke endpoint for trade GET ──────────────────────
	r.GET("/v1/trades/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade id"})
			return
		}
		t, err := di.Trade.GetTrade(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"trade_id":         t.ID(),
			"tenant_id":        t.TenantID(),
			"status":           t.Status(),
			"venue":            t.Venue(),
			"type":             t.Type(),
			"bought_currency":  t.BoughtCurrency(),
			"bought_amount":    t.BoughtAmount().String(),
			"sold_currency":    t.SoldCurrency(),
			"sold_amount":      t.SoldAmount().String(),
			"deal_rate":        t.DealRate().String(),
			"trade_date":       t.TradeDate(),
			"value_date":       t.ValueDate(),
			"version":          t.Version(),
		})
	})

	// ── Admin API (gated) ─────────────────────────────────────────────────
	// EXCHANGEOS_ENABLE_ADMIN_API=true exposes /v1/admin/{table} CRUD over
	// the full schema. Local dev: ON. Production: OFF unless explicit + scoped.
	adminRoutes := []string{}
	if os.Getenv("EXCHANGEOS_ENABLE_ADMIN_API") == "true" {
		if di.Pool == nil {
			logger.Warn("admin api requested but no postgres pool — backend is memory; admin routes NOT registered")
		} else {
			h := adminapi.NewHandler(di.Pool)
			h.Register(r)
			adminRoutes = []string{
				"/v1/admin/_schemas",
				"/v1/admin/:table",
				"/v1/admin/:table/:id",
			}
			logger.Info("admin api enabled", zap.Int("tables", len(adminapi.AllSchemas())))
		}
	}

	logger.Info("http routes registered",
		zap.Strings("routes", append([]string{
			"/healthz", "/readyz", "/version",
			"/v1/refdata/currencies", "/v1/trades/:id",
		}, adminRoutes...)),
	)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"nats/internal/context/logs"
	"nats/internal/context/metrics"
	"nats/internal/context/traces"
	"nats/internal/handler"
	"nats/internal/infra/nats"
	"nats/internal/infra/valkey"
	imiddle "nats/internal/middleware"
	"nats/internal/repo"
	"nats/internal/service"
	"nats/pkg/config"
	"nats/pkg/glogger"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		panic("config load failed")
	}

	glogger.GlobalLogger(cfg)
	metrics.StartMetrics()
	traces.StartTrace()
	logger, _ := logs.NewLogger(cfg.Log.Level)

	const apiVer = "v1"

	// NATS POOL Create and DI
	jsClient, err := nats.NewConnectionPool(ctx, cfg)
	if err != nil {
		glogger.Error(ctx, "JetStream connection failed", "error", err)
		os.Exit(1)
	}
	defer jsClient.ShutdownNatsPool(ctx)

	// Valkey Client Create
	valkeyClient, err := valkey.NewValkeyClient(ctx, cfg)
	if err != nil {
		glogger.Error(ctx, "ValkeyClient create failed", "error", err)
		jsClient.ShutdownNatsPool(ctx)
		os.Exit(1)
	}
	defer valkeyClient.Shutdown(ctx)

	// Repository resource create
	natsRepo := repo.NewNatsRepo(jsClient)
	valkeyRepo := repo.NewValkeyRepo(valkeyClient)

	// Service resource create
	ackDispatcher := service.NewAckDispatcher(100000, cfg.Publish.Worker, valkeyRepo) // Queue Size : TPS 100000
	ackDispatcher.Start()
	defer ackDispatcher.Stop()

	ackTimeout := 30 * time.Second
	publishSvc := service.NewPublishService(ackDispatcher, ackTimeout, natsRepo, valkeyRepo)
	topicSvc := service.NewTopicService(natsRepo, cfg)

	// Handler resource create
	accountBase := handler.AccountBaseHandlers(topicSvc)
	accountTopicBase := handler.AccountTopicBaseHandlers(topicSvc, publishSvc)

	// echo start
	e := echo.New()
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	imiddle.AttachMiddlewares(e, logger)

	// Setup router
	apiRouter := handler.NewApiRouter(accountBase, accountTopicBase)
	apiRouter.Register(e.Group(apiVer))

	go func() {
		glogger.Info(ctx, "API server is running", "url", "http://localhost:8080")
		if err := e.Start(":8080"); err != nil {
			glogger.Warn(ctx, "Server shutdown", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	glogger.Info(ctx, "Received server shutdown signal, cleaning up...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		glogger.Error(ctx, "Echo server shutdown failed", "error", err)
	}
	glogger.Info(ctx, "The server has been shut down normally")
}

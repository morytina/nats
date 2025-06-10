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
	natsrepo "nats/internal/infra/nats"
	"nats/internal/infra/valkey"
	imiddle "nats/internal/middleware"
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

	natsrepo.InitNatsPool(ctx, cfg)
	if err := valkey.InitValkeyClient(ctx, cfg); err != nil {
		natsrepo.ShutdownNatsPool(ctx)
		os.Exit(1)
	}

	e := echo.New()
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	imiddle.AttachMiddlewares(e, logger)

	topicSvc := service.NewTopicService()

	// ackDispatcher is futureAck after process gorutine
	ackDispatcher := service.NewAckDispatcher(100000, cfg.Publish.Worker) // Queue Size : TPS 100000
	ackDispatcher.Start()

	ackTimeout := 5 * time.Second

	// ✅ 변경된 부분: JetStreamClient 의존성 주입
	jsClient := natsrepo.GetJetStreamClient()
	publishSvc := service.NewPublishService(jsClient, ackDispatcher, ackTimeout)

	accountBase := handler.AccountBaseHandlers(topicSvc)
	accountTopicBase := handler.AccountTopicBaseHandlers(topicSvc, publishSvc)
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

	natsrepo.ShutdownNatsPool(ctx)
	valkey.ShutdownValkeyClient(ctx)
	ackDispatcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		glogger.Error(ctx, "Echo server shutdown failed", "error", err)
	}
	glogger.Info(ctx, "The server has been shut down normally")
}

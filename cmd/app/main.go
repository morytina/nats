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
		natsrepo.ShutdownNatsPool(ctx) // 생성된 커넥션 정리
		os.Exit(1)
	}

	e := echo.New()
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	imiddle.AttachMiddlewares(e, logger)

	topicSvc := service.NewTopicService()
	publishSvc := service.NewPublishService()

	handlers := handler.BuildHandlers(topicSvc, publishSvc)
	apiRouter := handler.NewApiRouter(handlers)
	apiRouter.Register(e.Group(apiVer))

	// 서버 시작
	go func() {
		glogger.Info(ctx, "API server is running", "url", "http://localhost:8080")
		if err := e.Start(":8080"); err != nil {
			glogger.Warn(ctx, "Server shutdown", "error", err)
		}
	}()

	// 종료 시그널 처리
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

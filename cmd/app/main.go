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
	emiddle "nats/internal/middleware"
	"nats/pkg/config"
	"nats/pkg/logger"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		panic("config load failed")
	}
	ctxlog, _ := logs.NewLogger("info")

	logger.InitLogger(cfg)
	metrics.StartMetrcis()
	traces.StartTrace()

	const apiVer = "v1"

	natsrepo.InitNatsPool(ctx, cfg)
	if err := valkey.InitValkeyClient(ctx, cfg); err != nil {
		natsrepo.ShutdownNatsPool(ctx) // 생성된 커넥션 정리
		os.Exit(1)
	}

	e := echo.New()
	emiddle.AttachMiddlewares(e, ctxlog)
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	apis := e.Group(apiVer)
	apis.Any("/", handler.ActionRouter) // 직접 핸들러로 분리 가능

	go func() {
		logger.Info(ctx, "API 서버 실행 중", "url", "http://localhost:8080")
		if err := e.Start(":8080"); err != nil {
			logger.Warn(ctx, "서버 종료", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info(ctx, "서버 종료 시그널 수신, 정리 중...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error(ctx, "Echo 서버 종료 실패", "error", err)
	}

	natsrepo.ShutdownNatsPool(ctx)
	valkey.ShutdownValkeyClient(ctx)

	logger.Info(ctx, "서버 정상 종료 완료")
}

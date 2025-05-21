package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"nats/internal/api"
	"nats/internal/config"
	icon "nats/internal/context"
	"nats/internal/logger"
	emiddle "nats/internal/middleware"
	"nats/internal/repository"
)

func main() {
	ctx := context.Background()
	icon.InitTracer()
	config.Init()
	logger.Init()
	repository.InitNatsPool(ctx)

	e := echo.New()
	emiddle.AttachMiddlewares(e)
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.Any("/", api.ActionRouter) // 직접 핸들러로 분리 가능

	go func() {
		if err := e.Start(":8080"); err != nil {
			logger.Warn(ctx, "서버 종료", "error", err)
		}
		logger.Info(ctx, "API 서버 실행 중", "url", "http://localhost:8080")
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

	repository.ShutdownNatsPool(ctx)

	logger.Info(ctx, "서버 정상 종료 완료")
}

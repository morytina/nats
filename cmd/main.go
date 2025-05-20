package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"

	"nats/internal/api"
	"nats/internal/config"
	icon "nats/internal/context"
	"nats/internal/logger"
	emiddle "nats/internal/middleware"
)

const (
	connectionNum = 3
)

var (
	jsPool    []nats.JetStreamContext
	ncPool    []*nats.Conn
	nextJSIdx uint32

	natsReconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "nats_reconnect_total", Help: "총 NATS 재연결 횟수"},
		[]string{"conn"},
	)
	natsDisconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "nats_disconnect_total", Help: "총 NATS 연결 끊김 횟수"},
		[]string{"conn"},
	)
)

func main() {
	icon.InitTracer()
	config.Init()
	logger.Init()

	prometheus.MustRegister(natsReconnects)
	prometheus.MustRegister(natsDisconnects)

	ncPool = make([]*nats.Conn, connectionNum)
	jsPool = make([]nats.JetStreamContext, connectionNum)

	ctx := context.Background()

	for i := 0; i < connectionNum; i++ {
		connName := fmt.Sprintf("SNS-API-Conn-%d", i)
		opts := makeNATSOptions(ctx, connName)

		nc, err := nats.Connect(nats.DefaultURL, opts...)
		if err != nil {
			logger.Fatal(ctx, "NATS 연결 실패", "index", i, "error", err)
		}
		ncPool[i] = nc

		js, err := nc.JetStream()
		if err != nil {
			logger.Fatal(ctx, "JetStream 컨텍스트 생성 실패", "index", i, "error", err)
		}
		jsPool[i] = js
	}

	e := echo.New()
	emiddle.AttachMiddlewares(e)

	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.Any("/", ActionRouter)

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

	for _, nc := range ncPool {
		if nc != nil && nc.IsConnected() {
			if err := nc.Drain(); err != nil {
				logger.Warn(ctx, "NATS 연결 종료 오류", "error", err)
			}
			nc.Close()
		}
	}
	logger.Info(ctx, "서버 정상 종료 완료")
}

func makeNATSOptions(ctx context.Context, connName string) []nats.Option {
	return []nats.Option{
		nats.Name(connName),
		nats.MaxReconnects(100),
		nats.ReconnectWait(2 * time.Second),
		nats.PingInterval(30 * time.Second),
		nats.MaxPingsOutstanding(3),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			natsReconnects.WithLabelValues(connName).Inc()
			logger.Info(ctx, "NATS 재연결 성공", "conn", connName, "url", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			natsDisconnects.WithLabelValues(connName).Inc()
			logger.Warn(ctx, "NATS 연결 끊김", "conn", connName, "error", err)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Error(ctx, "NATS 모든 재연결 실패", "conn", connName)
		}),
	}
}

func ActionRouter(c echo.Context) error {
	js := pickJetStream()
	action := c.QueryParam("Action")

	// ⚠️ 개발용: trace 시작
	reqCtx := c.Request().Context()
	tr := otel.Tracer("sns-api")
	ctx, span := tr.Start(reqCtx, action)
	defer span.End()

	// 요청에 새로운 context 주입 (핵심)
	c.SetRequest(c.Request().WithContext(ctx))

	switch action {
	case "createTopic":
		return api.CreateTopicHandler(js)(c)
	case "deleteTopic":
		return api.DeleteTopicHandler(js)(c)
	case "listTopics":
		logger.Info(ctx, "listTopics 호출")
		return api.ListTopicsHandler(js)(c)
	case "publish":
		return api.PublishHandler(js)(c)
	default:
		return c.String(400, "invalid Action")
	}
}

func pickJetStream() nats.JetStreamContext {
	ctx := context.Background()
	for i := 0; i < connectionNum; i++ {
		idx := int(atomic.AddUint32(&nextJSIdx, 1)) % connectionNum
		nc := ncPool[idx]
		js := jsPool[idx]

		if nc == nil || nc.IsClosed() || !nc.IsConnected() {
			logger.Warn(ctx, "JetStream 연결 비정상", "index", idx)
			connName := fmt.Sprintf("SNS-API-Conn-%d", idx)
			opts := makeNATSOptions(ctx, connName)

			newNc, err := nats.Connect(nats.DefaultURL, opts...)
			if err != nil {
				logger.Error(ctx, "재연결 실패", "index", idx, "error", err)
				continue
			}
			ncPool[idx] = newNc

			newJs, err := newNc.JetStream()
			if err != nil {
				logger.Error(ctx, "JetStreamContext 재생성 실패", "index", idx, "error", err)
				continue
			}
			jsPool[idx] = newJs
			logger.Info(ctx, "JetStream 재연결 성공", "index", idx)
			return newJs
		}
		return js
	}
	logger.Error(ctx, "사용 가능한 JetStream 연결 없음, 기본 연결 반환")
	return jsPool[0]
}

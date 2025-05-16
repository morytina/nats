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
	"github.com/labstack/echo/v4/middleware"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"nats/internal/api"
	"nats/internal/logger"
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
	logger.Init()

	prometheus.MustRegister(natsReconnects)
	prometheus.MustRegister(natsDisconnects)

	ncPool = make([]*nats.Conn, connectionNum)
	jsPool = make([]nats.JetStreamContext, connectionNum)

	for i := 0; i < connectionNum; i++ {
		connName := fmt.Sprintf("SNS-API-Conn-%d", i)
		opts := makeNATSOptions(connName)

		nc, err := nats.Connect(nats.DefaultURL, opts...)
		if err != nil {
			logger.Fatal("NATS 연결 실패", "index", i, "error", err)
		}
		ncPool[i] = nc

		js, err := nc.JetStream()
		if err != nil {
			logger.Fatal("JetStream 컨텍스트 생성 실패", "index", i, "error", err)
		}
		jsPool[i] = js
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.Any("/", ActionRouter)

	go func() {
		logger.Info("API 서버 실행 중", "url", "http://localhost:8080")
		if err := e.Start(":8080"); err != nil {
			logger.Warn("서버 종료", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("서버 종료 시그널 수신, 정리 중...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Echo 서버 종료 실패", "error", err)
	}

	for _, nc := range ncPool {
		if nc != nil && nc.IsConnected() {
			if err := nc.Drain(); err != nil {
				logger.Warn("NATS 연결 종료 오류", "error", err)
			}
			nc.Close()
		}
	}
	logger.Info("서버 정상 종료 완료")
}

func makeNATSOptions(connName string) []nats.Option {
	return []nats.Option{
		nats.Name(connName),
		nats.MaxReconnects(100),
		nats.ReconnectWait(2 * time.Second),
		nats.PingInterval(30 * time.Second),
		nats.MaxPingsOutstanding(3),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			natsReconnects.WithLabelValues(connName).Inc()
			logger.Info("NATS 재연결 성공", "conn", connName, "url", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			natsDisconnects.WithLabelValues(connName).Inc()
			logger.Warn("NATS 연결 끊김", "conn", connName, "error", err)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Error("NATS 모든 재연결 실패", "conn", connName)
		}),
	}
}

func ActionRouter(c echo.Context) error {
	js := pickJetStream()
	action := c.QueryParam("Action")

	switch action {
	case "createTopic":
		return api.CreateTopicHandler(js)(c)
	case "deleteTopic":
		return api.DeleteTopicHandler(js)(c)
	case "listTopics":
		return api.ListTopicsHandler(js)(c)
	case "publish":
		return api.PublishHandler(js)(c)
	default:
		return c.String(400, "invalid Action")
	}
}

func pickJetStream() nats.JetStreamContext {
	for i := 0; i < connectionNum; i++ {
		idx := int(atomic.AddUint32(&nextJSIdx, 1)) % connectionNum
		nc := ncPool[idx]
		js := jsPool[idx]

		if nc == nil || nc.IsClosed() || !nc.IsConnected() {
			logger.Warn("JetStream 연결 비정상", "index", idx)
			connName := fmt.Sprintf("SNS-API-Conn-%d", idx)
			opts := makeNATSOptions(connName)

			newNc, err := nats.Connect(nats.DefaultURL, opts...)
			if err != nil {
				logger.Error("재연결 실패", "index", idx, "error", err)
				continue
			}
			ncPool[idx] = newNc

			newJs, err := newNc.JetStream()
			if err != nil {
				logger.Error("JetStreamContext 재생성 실패", "index", idx, "error", err)
				continue
			}
			jsPool[idx] = newJs
			logger.Info("JetStream 재연결 성공", "index", idx)
			return newJs
		}
		return js
	}
	logger.Error("사용 가능한 JetStream 연결 없음, 기본 연결 반환")
	return jsPool[0]
}

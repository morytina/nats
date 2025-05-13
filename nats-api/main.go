package main

import (
	"context"
	"fmt"
	"log"
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
)

const (
	cspRegion     = "kr-cp-1"
	cspAccount    = "100000000000"
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
	prometheus.MustRegister(natsReconnects)
	prometheus.MustRegister(natsDisconnects)

	ncPool = make([]*nats.Conn, connectionNum)
	jsPool = make([]nats.JetStreamContext, connectionNum)

	for i := 0; i < connectionNum; i++ {
		connName := fmt.Sprintf("SNS-API-Conn-%d", i)
		opts := makeNATSOptions(connName)

		nc, err := nats.Connect(nats.DefaultURL, opts...)
		if err != nil {
			log.Fatalf("[ERROR] NATS 연결 실패 (%d): %v", i, err)
		}
		ncPool[i] = nc

		js, err := nc.JetStream()
		if err != nil {
			log.Fatalf("[ERROR] JetStream 컨텍스트 생성 실패 (%d): %v", i, err)
		}
		jsPool[i] = js
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.Any("/", ActionRouter)

	go func() {
		log.Println("[INFO] API 서버 실행 중: http://localhost:8080")
		if err := e.Start(":8080"); err != nil {
			log.Printf("[WARN] 서버 종료: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("[INFO] 서버 종료 시그널 수신, 정리 중...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Echo 서버 종료 실패: %v", err)
	}

	for _, nc := range ncPool {
		if nc != nil && nc.IsConnected() {
			if err := nc.Drain(); err != nil {
				log.Printf("[WARN] NATS 연결 종료 오류: %v", err)
			}
			nc.Close()
		}
	}
	log.Println("[INFO] 서버 정상 종료 완료")
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
			log.Printf("[INFO] [%s] 재연결 성공: %s", connName, nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			natsDisconnects.WithLabelValues(connName).Inc()
			log.Printf("[WARN] [%s] 연결 끊김: %v", connName, err)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("[ERROR] [%s] 모든 재연결 실패. 연결 종료됨.", connName)
		}),
	}
}

func ActionRouter(c echo.Context) error {
	js := pickJetStream()
	action := c.QueryParam("Action")

	switch action {
	case "createTopic":
		return createTopicHandler(js)(c)
	case "deleteTopic":
		return deleteTopicHandler(js)(c)
	case "listTopics":
		return listTopicsHandler(js)(c)
	case "publish":
		return publishHandler(js)(c)
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
			log.Printf("[WARN] 연결 %d 비정상, 재연결 시도 중...", idx)
			connName := fmt.Sprintf("SNS-API-Conn-%d", idx)
			opts := makeNATSOptions(connName)

			newNc, err := nats.Connect(nats.DefaultURL, opts...)
			if err != nil {
				log.Printf("[ERROR] 연결 %d 재연결 실패: %v", idx, err)
				continue
			}
			ncPool[idx] = newNc

			newJs, err := newNc.JetStream()
			if err != nil {
				log.Printf("[ERROR] JetStreamContext 재생성 실패: %v", err)
				continue
			}
			jsPool[idx] = newJs

			log.Printf("[INFO] 연결 %d 재연결 성공", idx)
			return newJs
		}
		return js
	}

	log.Println("[ERROR] 사용 가능한 JetStream 연결 없음. 기본 연결 사용")
	return jsPool[0]
}

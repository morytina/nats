package nats

import (
	"context"
	"fmt"
	"nats/pkg/config"
	"nats/pkg/logger"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	natsReconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "nats_reconnect_total", Help: "총 NATS 재연결 횟수"},
		[]string{"conn"},
	)
	natsDisconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "nats_disconnect_total", Help: "총 NATS 연결 실패 횟수"},
		[]string{"conn"},
	)
)

func InitNatsPool(ctx context.Context) {
	if pool = config.Root.Nats.ConnPoolCnt; pool == 0 {
		logger.Warn(ctx, "Connection count is 0. Setting default 3.", "pool size", pool)
		pool = 3
	}
	ncPool = make([]*nats.Conn, pool)
	jsPool = make([]nats.JetStreamContext, pool)

	prometheus.MustRegister(natsReconnects)
	prometheus.MustRegister(natsDisconnects)

	for i := 0; i < pool; i++ {
		connName := fmt.Sprintf("SNS-API-Conn-%d", i)
		opts := makeNATSOptions(ctx, connName)

		nc, err := nats.Connect(nats.DefaultURL, opts...)
		if err != nil {
			logger.Fatal(ctx, "NATS 연결 실패", "index", i, "error", err)
		}
		ncPool[i] = nc

		js, err := nc.JetStream(nats.PublishAsyncMaxPending(100000))
		if err != nil {
			logger.Fatal(ctx, "JetStream 사용 실패", "index", i, "error", err)
		}
		jsPool[i] = js
	}
	logger.Info(ctx, "NATS POOL 생성 성공", "pool", pool)

	SetJetStreamClient(&defaultJetStreamClient{}) // 인터페이스 주입
}

func ShutdownNatsPool(ctx context.Context) {
	for i, nc := range ncPool {
		if nc != nil && nc.IsConnected() {
			if err := nc.Drain(); err != nil {
				logger.Warn(ctx, "NATS 연결 종료 오류", "index", i, "error", err)
			}
			nc.Close()
			logger.Info(ctx, "NATS 연결 종료 완료", "index", i)
		}
	}
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
			logger.Info(ctx, "NATS 재연결", "conn", connName, "url", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			natsDisconnects.WithLabelValues(connName).Inc()
			logger.Warn(ctx, "NATS 연결 실패", "conn", connName, "error", err)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Error(ctx, "NATS 모든 재연결 실패", "conn", connName)
		}),
	}
}

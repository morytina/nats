package nats

import (
	"context"
	"fmt"
	"nats/internal/context/metrics"
	"nats/pkg/config"
	"nats/pkg/glogger"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func InitNatsPool(ctx context.Context, cfg *config.Config) {
	if pool = cfg.Nats.ConnPoolCnt; pool == 0 {
		glogger.Warn(ctx, "Connection count is 0. Setting default 3.", "pool size", pool)
		pool = 3
	}
	ncPool = make([]*nats.Conn, pool)
	jsPool = make([]jetstream.JetStream, pool)
	for i := 0; i < pool; i++ {
		connName := fmt.Sprintf("SNS-API-Conn-%d", i)
		opts := makeNATSOptions(ctx, connName)

		nc, err := nats.Connect(nats.DefaultURL, opts...)
		if err != nil {
			glogger.Fatal(ctx, "NATS 연결 실패", "index", i, "error", err)
		}
		ncPool[i] = nc

		js, err := jetstream.New(nc, jetstream.WithPublishAsyncMaxPending(100000)) // nc.JetStream(nats.PublishAsyncMaxPending(100000))
		if err != nil {
			glogger.Fatal(ctx, "JetStream 사용 실패", "index", i, "error", err)
		}
		jsPool[i] = js
	}
	glogger.Info(ctx, "NATS POOL 생성 성공", "pool", pool)

	SetJetStreamClient(&defaultJetStreamClient{}) // 인터페이스 주입
}

func ShutdownNatsPool(ctx context.Context) {
	for i, nc := range ncPool {
		if nc != nil && nc.IsConnected() {
			if err := nc.Drain(); err != nil {
				glogger.Warn(ctx, "NATS 연결 종료 오류", "index", i, "error", err)
			}
			nc.Close()
			glogger.Info(ctx, "NATS 연결 종료 완료", "index", i)
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
			metrics.NatsReconnects.WithLabelValues(connName).Inc()
			glogger.Info(ctx, "NATS 재연결", "conn", connName, "url", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			metrics.NatsDisconnects.WithLabelValues(connName).Inc()
			glogger.Warn(ctx, "NATS 연결 실패", "conn", connName, "error", err)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			glogger.Error(ctx, "NATS 모든 재연결 실패", "conn", connName)
		}),
	}
}

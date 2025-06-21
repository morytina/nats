package nats

import (
	"context"
	"fmt"
	"nats/internal/context/metrics"
	"nats/pkg/config"
	"nats/pkg/glogger"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetStreamPool interface {
	GetJetStream(ctx context.Context) jetstream.JetStream
}

type ConnectionPool struct {
	ncPool  []*nats.Conn
	jsPool  []jetstream.JetStream
	nextIdx uint32
	size    int
}

// NewConnectionPool creates a pool of JetStream connections
func NewConnectionPool(ctx context.Context, cfg *config.Config) (*ConnectionPool, error) {
	poolSize := cfg.Nats.ConnPoolCnt
	if pool := cfg.Nats.ConnPoolCnt; pool == 0 {
		glogger.Warn(ctx, "Connection count is 0. Setting default 3.", "pool size", poolSize)
		poolSize = 3
	}

	ncPool := make([]*nats.Conn, poolSize)
	jsPool := make([]jetstream.JetStream, poolSize)

	for i := 0; i < poolSize; i++ {
		connName := fmt.Sprintf("SNS-API-Conn-%d", i)
		opts := makeNATSOptions(ctx, connName)

		nc, err := nats.Connect(nats.DefaultURL, opts...)
		if err != nil {
			return nil, fmt.Errorf("NATS 연결 실패 index=%d: %w", i, err)
		}
		ncPool[i] = nc

		js, err := jetstream.New(nc, jetstream.WithPublishAsyncMaxPending(100000))
		if err != nil {
			return nil, fmt.Errorf("JetStream 사용 실패 index=%d: %w", i, err)
		}
		jsPool[i] = js
	}

	glogger.Info(ctx, "NATS POOL 생성 성공", "pool", poolSize)
	return &ConnectionPool{
		ncPool: ncPool,
		jsPool: jsPool,
		size:   poolSize,
	}, nil
}

// GetJetStream selects an available JetStream client from the pool (with reconnect if needed)
func (c *ConnectionPool) GetJetStream(ctx context.Context) jetstream.JetStream {
	for i := 0; i < c.size; i++ {
		idx := int(atomic.AddUint32(&c.nextIdx, 1)) % c.size
		nc := c.ncPool[idx]
		js := c.jsPool[idx]

		if nc == nil || nc.IsClosed() || !nc.IsConnected() {
			glogger.Warn(ctx, "JetStream 연결 문제", "index", idx)
			connName := fmt.Sprintf("SNS-API-Conn-%d", idx)
			opts := makeNATSOptions(ctx, connName)

			newNc, err := nats.Connect(nats.DefaultURL, opts...)
			if err != nil {
				glogger.Error(ctx, "재연결 실패", "index", idx, "error", err)
				continue
			}
			newJs, err := jetstream.New(newNc)
			if err != nil {
				glogger.Error(ctx, "JetStreamContext 재생성 실패", "index", idx, "error", err)
				continue
			}
			c.ncPool[idx] = newNc
			c.jsPool[idx] = newJs
			glogger.Info(ctx, "JetStream 재연결 성공", "index", idx)
			return newJs
		}
		return js
	}

	glogger.Error(ctx, "사용 가능한 JetStream 연결 없음")
	return c.jsPool[0]
}

// ShutdownNatsPool gracefully closes all NATS connections
func (c *ConnectionPool) ShutdownNatsPool(ctx context.Context) {
	for i, nc := range c.ncPool {
		if nc != nil && nc.IsConnected() {
			if err := nc.Drain(); err != nil {
				glogger.Warn(ctx, "NATS 연결 종료 오류", "index", i, "error", err)
			}
			nc.Close()
			glogger.Info(ctx, "NATS 연결 종료 완료", "index", i)
		}
	}
}

// makeNATSOptions builds common connection options
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

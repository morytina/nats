package nats

import (
	"context"
	"errors"
	"fmt"
	"nats/internal/context/logs"
	"nats/internal/context/metrics"
	"nats/pkg/config"
	"nats/pkg/glogger"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type JetStreamPool interface {
	GetJetStream(ctx context.Context) (jetstream.JetStream, error)
	ShutdownNatsPool(ctx context.Context)
}

type connectionPool struct {
	ncPool  []*nats.Conn
	jsPool  []jetstream.JetStream
	nextIdx uint32
	size    int
}

// NewConnectionPool creates a pool of JetStream connections
func NewConnectionPool(ctx context.Context, cfg *config.Config) (JetStreamPool, error) {
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
	return &connectionPool{
		ncPool: ncPool,
		jsPool: jsPool,
		size:   poolSize,
	}, nil
}

// GetJetStream selects an available JetStream client from the pool (with reconnect if needed)
func (c *connectionPool) GetJetStream(ctx context.Context) (jetstream.JetStream, error) {
	for i := 0; i < c.size; i++ {
		idx := int(atomic.AddUint32(&c.nextIdx, 1)) % c.size
		nc := c.ncPool[idx]
		js := c.jsPool[idx]

		if nc == nil || nc.IsClosed() || !nc.IsConnected() {
			logs.GetLogger(ctx).Warn("JetStream 연결 문제", logs.WithTraceFields(ctx, zap.Int("index", idx))...)
			connName := fmt.Sprintf("SNS-API-Conn-%d", idx)
			opts := makeNATSOptions(ctx, connName)

			newNc, err := nats.Connect(nats.DefaultURL, opts...)
			if err != nil {
				logs.GetLogger(ctx).Error("재연결 실패", logs.WithTraceFields(ctx, zap.Int("index", idx), zap.Error(err))...)
				continue
			}
			newJs, err := jetstream.New(newNc)
			if err != nil {
				logs.GetLogger(ctx).Error("JetStreamContext 재생성 실패", logs.WithTraceFields(ctx, zap.Int("index", idx), zap.Error(err))...)
				continue
			}
			c.ncPool[idx] = newNc
			c.jsPool[idx] = newJs
			logs.GetLogger(ctx).Info("JetStream 재연결 성공", logs.WithTraceFields(ctx, zap.Int("index", idx))...)
			return newJs, nil
		}
		return js, nil
	}

	err := errors.New("no available JetStream connection")
	logs.GetLogger(ctx).Error("GetJetStream fail", logs.WithTraceFields(ctx, zap.Error(err))...)
	return nil, err
}

// ShutdownNatsPool gracefully closes all NATS connections
func (c *connectionPool) ShutdownNatsPool(ctx context.Context) {
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

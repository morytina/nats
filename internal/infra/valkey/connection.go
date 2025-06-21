package valkey

import (
	"context"
	"time"

	"nats/internal/context/metrics"
	"nats/pkg/config"
	"nats/pkg/glogger"

	"github.com/go-redis/redis/v8"
)

func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	addr := cfg.Valkey.Addr
	password := cfg.Valkey.Password
	db := cfg.Valkey.DB

	rawClient := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		MinIdleConns: 3,
		PoolSize:     100,
	})

	if err := rawClient.Ping(ctx).Err(); err != nil {
		metrics.ValkeyFailures.Inc()
		glogger.Error(ctx, "Valkey 연결 실패", "addr", addr, "error", err)
		return nil, err
	}

	metrics.ValkeyReconnects.Inc()
	glogger.Info(ctx, "Valkey 연결 성공", "addr", addr)

	return NewRedisClient(rawClient), nil
}

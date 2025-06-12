package valkey

import (
	"context"
	"time"

	"nats/internal/context/metrics"
	"nats/pkg/config"
	"nats/pkg/glogger"

	"github.com/go-redis/redis/v8"
)

var clientInstance Client // 인터페이스로 선언

func InitValkeyClient(ctx context.Context, cfg *config.Config) error {
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
		return err
	}

	metrics.ValkeyReconnects.Inc()
	glogger.Info(ctx, "Valkey 연결 성공", "addr", addr)

	// 인터페이스 구현체로 등록
	clientInstance = NewRedisClient(rawClient)
	return nil
}

func GetClient() Client {
	return clientInstance
}

func ShutdownValkeyClient(ctx context.Context) {
	if rc, ok := clientInstance.(*redisClient); ok {
		if err := rc.raw.Close(); err != nil {
			glogger.Warn(ctx, "Valkey 종료 오류", "error", err)
		} else {
			glogger.Info(ctx, "Valkey 클라이언트 정상 종료")
		}
	}
}

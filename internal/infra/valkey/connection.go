package valkey

import (
	"context"
	"time"

	"nats/pkg/config"
	"nats/pkg/logger"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	clientInstance Client // 인터페이스로 선언

	valkeyReconnects = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "valkey_reconnect_total",
			Help: "총 Valkey 재연결 횟수",
		},
	)
	valkeyFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "valkey_connection_failures_total",
			Help: "Valkey 연결 실패 횟수",
		},
	)
)

func InitValkeyClient(ctx context.Context) {
	prometheus.MustRegister(valkeyReconnects)
	prometheus.MustRegister(valkeyFailures)

	addr := config.Root.Valkey.Addr
	password := config.Root.Valkey.Password
	db := config.Root.Valkey.DB

	rawClient := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		MinIdleConns: 3,
		PoolSize:     10,
	})

	if err := rawClient.Ping(ctx).Err(); err != nil {
		valkeyFailures.Inc()
		logger.Fatal(ctx, "Valkey 연결 실패", "addr", addr, "error", err)
	}

	valkeyReconnects.Inc()
	logger.Info(ctx, "Valkey 연결 성공", "addr", addr)

	// 인터페이스 구현체로 등록
	clientInstance = NewRedisClient(rawClient)
}

func GetClient() Client {
	return clientInstance
}

func ShutdownValkeyClient(ctx context.Context) {
	if rc, ok := clientInstance.(*redisClient); ok {
		if err := rc.raw.Close(); err != nil {
			logger.Warn(ctx, "Valkey 종료 오류", "error", err)
		} else {
			logger.Info(ctx, "Valkey 클라이언트 정상 종료")
		}
	}
}

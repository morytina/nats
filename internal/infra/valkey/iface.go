package valkey

import (
	"context"
	"time"
)

type Client interface {
	SetKeyWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetKey(ctx context.Context, key string) (string, error)
	DelKey(ctx context.Context, keys ...string) error
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SRem(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	Shutdown(ctx context.Context) error
}

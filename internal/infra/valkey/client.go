package valkey

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisClient struct {
	raw *redis.Client
}

func NewRedisClient(raw *redis.Client) Client {
	return &redisClient{raw: raw}
}

func (c *redisClient) SetKeyWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.raw.Set(ctx, key, value, ttl).Err()
}

func (c *redisClient) GetKey(ctx context.Context, key string) (string, error) {
	return c.raw.Get(ctx, key).Result()
}

func (c *redisClient) DelKey(ctx context.Context, keys ...string) error {
	return c.raw.Del(ctx, keys...).Err()
}

func (c *redisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.raw.SAdd(ctx, key, members...).Err()
}

func (c *redisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.raw.SRem(ctx, key, members...).Err()
}

func (c *redisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.raw.SMembers(ctx, key).Result()
}

func (c *redisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.raw.HSet(ctx, key, values...).Err()
}

func (c *redisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return c.raw.HGet(ctx, key, field).Result()
}

func (c *redisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.raw.HGetAll(ctx, key).Result()
}

func (c *redisClient) Shutdown(ctx context.Context) error {
	return c.raw.Close()
}

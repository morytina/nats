package valkey

import (
	"context"
	"time"

	"nats/pkg/config"

	"github.com/valkey-io/valkey-go"
)

type ValkeyClient interface {
	Shutdown(ctx context.Context)
	GetValue(ctx context.Context, key string) (string, error)
	SetValueWithTTL(ctx context.Context, key string, value string, ttl time.Duration) error
}

type valkeyClient struct {
	client valkey.Client
}

func NewValkeyClient(ctx context.Context, cfg *config.Config) (ValkeyClient, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Valkey.Addr},
		Password:    cfg.Valkey.Password,
	})
	if err != nil {
		return nil, err
	}
	return &valkeyClient{client: client}, nil
}

// graceful shutdown (e.g., when main.go ends)
func (v *valkeyClient) Shutdown(ctx context.Context) {
	v.client.Close()
}

func (v *valkeyClient) GetValue(ctx context.Context, key string) (string, error) {
	return v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
}

func (v *valkeyClient) SetValueWithTTL(ctx context.Context, key string, value string, ttl time.Duration) error {
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(value).Ex(ttl).Build()).Error()
}

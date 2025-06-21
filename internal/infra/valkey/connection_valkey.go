package valkey

import (
	"context"

	"nats/pkg/config"

	"github.com/valkey-io/valkey-go"
)

type ValkeyClient struct {
	client valkey.Client
}

func NewValkeyClient(ctx context.Context, cfg *config.Config) (*ValkeyClient, error) {
	addr := cfg.Valkey.Addr
	password := cfg.Valkey.Password

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
		Password:    password,
	})

	if err != nil {
		return nil, err
	}

	return &ValkeyClient{client: client}, nil
}

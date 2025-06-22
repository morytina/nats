package repo

import (
	"context"
	"encoding/json"
	"nats/internal/context/logs"
	"nats/internal/entity"
	"nats/internal/infra/valkey"
	"time"

	"go.uber.org/zap"
)

type ValkeyRepo interface {
	StoreAckResult(ctx context.Context, id string, result entity.AckResult) error
	GetAckStatus(ctx context.Context, id string) (string, error)
}

type valkeyRepo struct {
	valkeyClient valkey.ValkeyClient
}

func NewValkeyRepo(valkeyClient valkey.ValkeyClient) ValkeyRepo {
	return &valkeyRepo{valkeyClient: valkeyClient}
}

func (s *valkeyRepo) StoreAckResult(ctx context.Context, id string, result entity.AckResult) error {
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = s.valkeyClient.SetValueWithTTL(ctx, id, string(bytes), 30*time.Second)
	if err != nil {
		logs.GetLogger(ctx).Warn("Failed to save ACK status", zap.String("id", id), zap.Error(err))
	}
	return err
}

func (s *valkeyRepo) GetAckStatus(ctx context.Context, id string) (string, error) {
	return s.valkeyClient.GetValue(ctx, id)
}

package repo

import (
	"context"
	"encoding/json"
	"nats/internal/context/logs"
	"nats/internal/entity"
	"nats/internal/infra/nats"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/valkey-io/valkey-go"
	"go.uber.org/zap"
)

type PublishRepo interface {
	StoreAckResult(ctx context.Context, id string, result entity.AckResult) error
	GetAckStatus(ctx context.Context, id string) (string, error)
	PublishAsyncMessage(ctx context.Context, message, subject string) (jetstream.PubAckFuture, error)
}

type publishRepo struct {
	jsClient     nats.JetStreamPool
	valkeyClient valkey.Client
}

func NewPublishRepo(jsClient nats.JetStreamPool, valkeyClient valkey.Client) PublishRepo {
	return &publishRepo{jsClient: jsClient, valkeyClient: valkeyClient}
}

func (s *publishRepo) PublishAsyncMessage(ctx context.Context, message, subject string) (jetstream.PubAckFuture, error) {
	js := s.jsClient.GetJetStream(ctx)
	return js.PublishAsync(subject, []byte(message))
}

func (s *publishRepo) StoreAckResult(ctx context.Context, id string, result entity.AckResult) error {
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = s.valkeyClient.Do(ctx, s.valkeyClient.B().Set().Key(id).Value(string(bytes)).Ex(30*time.Second).Build()).Error()
	if err != nil {
		logs.GetLogger(ctx).Warn("Failed to save ACK status", zap.String("id", id), zap.Error(err))
	}
	return err
}

func (s *publishRepo) GetAckStatus(ctx context.Context, id string) (string, error) {
	return s.valkeyClient.Do(ctx, s.valkeyClient.B().Get().Key(id).Build()).ToString()
}

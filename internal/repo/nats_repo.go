package repo

import (
	"context"

	"nats/internal/infra/nats"

	"github.com/nats-io/nats.go/jetstream"
)

type NatsRepo interface {
	PublishAsyncMessage(ctx context.Context, message, subject string) (jetstream.PubAckFuture, error)

	CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error)
	DeleteStream(ctx context.Context, name string) error
	ListStreamNames(ctx context.Context) (<-chan string, error)
}

type natsRepo struct {
	jsClient nats.JetStreamPool
}

func NewNatsRepo(jsClient nats.JetStreamPool) NatsRepo {
	return &natsRepo{jsClient: jsClient}
}

func (s *natsRepo) PublishAsyncMessage(ctx context.Context, message, subject string) (jetstream.PubAckFuture, error) {
	js := s.jsClient.GetJetStream(ctx)
	return js.PublishAsync(subject, []byte(message))
}

func (s *natsRepo) CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	js := s.jsClient.GetJetStream(ctx)
	return js.CreateStream(ctx, cfg)
}

func (s *natsRepo) DeleteStream(ctx context.Context, name string) error {
	js := s.jsClient.GetJetStream(ctx)
	return js.DeleteStream(ctx, name)
}

func (s *natsRepo) ListStreamNames(ctx context.Context) (<-chan string, error) {
	js := s.jsClient.GetJetStream(ctx)
	lister := js.StreamNames(ctx)
	return lister.Name(), nil
}

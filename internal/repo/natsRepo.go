package repo

import (
	"context"
	"time"

	"nats/internal/infra/nats"

	"github.com/nats-io/nats.go/jetstream"
)

type NatsRepo interface {
	PublishAsyncMessage(ctx context.Context, message, subject string) (jetstream.PubAckFuture, error)

	CreateStream(ctx context.Context, name string) (jetstream.Stream, error)
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
	js, err := s.jsClient.GetJetStream(ctx)
	if err != nil {
		return nil, err
	}
	return js.PublishAsync(subject, []byte(message))
}

func (s *natsRepo) CreateStream(ctx context.Context, name string) (jetstream.Stream, error) {
	streamCfg := jetstream.StreamConfig{
		Name:              name,
		Subjects:          []string{name},
		Storage:           jetstream.FileStorage,
		Replicas:          1,
		Retention:         jetstream.LimitsPolicy,
		Discard:           jetstream.DiscardOld,
		MaxMsgs:           -1,
		MaxMsgsPerSubject: -1,
		MaxBytes:          -1,
		MaxAge:            96 * time.Hour,
		MaxMsgSize:        262144,
		Duplicates:        0,
		AllowRollup:       false,
		DenyDelete:        false,
		DenyPurge:         false,
	}

	js, err := s.jsClient.GetJetStream(ctx)
	if err != nil {
		return nil, err
	}
	return js.CreateStream(ctx, streamCfg)
}

func (s *natsRepo) DeleteStream(ctx context.Context, name string) error {
	js, err := s.jsClient.GetJetStream(ctx)
	if err != nil {
		return err
	}
	return js.DeleteStream(ctx, name)
}

func (s *natsRepo) ListStreamNames(ctx context.Context) (<-chan string, error) {
	js, err := s.jsClient.GetJetStream(ctx)
	if err != nil {
		return nil, err
	}
	lister := js.StreamNames(ctx)
	return lister.Name(), nil
}

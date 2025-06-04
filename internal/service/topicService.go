package service

import (
	"context"
	"time"

	"nats/internal/context/logs"
	"nats/internal/context/traces"
	natsrepo "nats/internal/infra/nats"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type TopicService interface {
	CreateTopic(ctx context.Context, name, subject string) error
	DeleteTopic(ctx context.Context, name string) error
	ListTopics(ctx context.Context) ([]string, error)
}

type topicService struct{}

func NewTopicService() TopicService {
	return &topicService{}
}

func (s *topicService) CreateTopic(ctx context.Context, name, subject string) error {
	js := natsrepo.GetJetStream(ctx)

	streamCfg := &nats.StreamConfig{
		Name:              name,
		Subjects:          []string{subject},
		Storage:           nats.FileStorage,
		Replicas:          1,
		Retention:         nats.LimitsPolicy,
		Discard:           nats.DiscardOld,
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

	_, err := js.AddStream(streamCfg)
	return err
}

func (s *topicService) DeleteTopic(ctx context.Context, name string) error {
	js := natsrepo.GetJetStream(ctx)
	return js.DeleteStream(name)
}

func (s *topicService) ListTopics(ctx context.Context) ([]string, error) {
	ctx, span := traces.StartSpan(ctx, "listTopics")
	defer span.End()

	logs.GetLogger(ctx).Info("ListTopics trace test", logs.WithTraceFields(ctx, zap.String("phase", "list"))...)

	js := natsrepo.GetJetStream(ctx)

	var topics []string
	lister := js.StreamNames()
	for name := range lister {
		topics = append(topics, "srn:scp:sns:kr-cp-1:100000000000:"+name)
	}
	return topics, nil
}

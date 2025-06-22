package service

import (
	"context"
	"nats/internal/context/logs"
	"nats/internal/context/traces"
	"nats/internal/repo"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type TopicService interface {
	CreateTopic(ctx context.Context, name, subject string) error
	DeleteTopic(ctx context.Context, name string) error
	ListTopics(ctx context.Context) ([]string, error)
}

type topicService struct {
	natsRepo repo.NatsRepo
}

func NewTopicService(natsRepo repo.NatsRepo) TopicService {
	return &topicService{natsRepo: natsRepo}
}

func (s *topicService) CreateTopic(ctx context.Context, name, subject string) error {
	streamCfg := jetstream.StreamConfig{
		Name:              name,
		Subjects:          []string{subject},
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

	_, err := s.natsRepo.CreateStream(ctx, streamCfg)
	return err
}

func (s *topicService) DeleteTopic(ctx context.Context, name string) error {
	return s.natsRepo.DeleteStream(ctx, name)
}

func (s *topicService) ListTopics(ctx context.Context) ([]string, error) {
	ctx, span := traces.StartSpan(ctx, "listTopics")
	defer span.End()

	logs.GetLogger(ctx).Info("ListTopics trace test", logs.WithTraceFields(ctx, zap.String("phase", "list"))...)

	namesCh, err := s.natsRepo.ListStreamNames(ctx)
	if err != nil {
		return nil, err
	}

	var topics []string
	for name := range namesCh {
		topics = append(topics, "srn:scp:sns:kr-cp-1:100000000000:"+name)
	}
	return topics, nil
}

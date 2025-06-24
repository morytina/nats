package service

import (
	"context"
	"nats/internal/context/logs"
	"nats/internal/context/traces"
	"nats/internal/entity"
	"nats/internal/repo"
	"nats/pkg/config"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type TopicService interface {
	CreateTopic(ctx context.Context, name, subject string) error
	DeleteTopic(ctx context.Context, name string) error
	ListTopics(ctx context.Context, account string) ([]entity.Topic, error)
}

type topicService struct {
	natsRepo repo.NatsRepo
	cfg      *config.Config
}

func NewTopicService(natsRepo repo.NatsRepo, cfg *config.Config) TopicService {
	return &topicService{natsRepo: natsRepo, cfg: cfg}
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

func (s *topicService) ListTopics(ctx context.Context, account string) ([]entity.Topic, error) {
	ctx, span := traces.StartSpan(ctx, "listTopics")
	defer span.End()

	logs.GetLogger(ctx).Info("ListTopics trace test", logs.WithTraceFields(ctx, zap.String("phase", "list"))...)

	namesCh, err := s.natsRepo.ListStreamNames(ctx)
	if err != nil {
		traces.RecordSpanError(ctx, span, "natsRepo.ListStreamNames error", err)
		return nil, err
	}

	var topics []entity.Topic
	for name := range namesCh {
		var sb strings.Builder
		sb.Grow(len("srn:scp:sns:::") + len(s.cfg.Region) + len(account) + len(name))
		sb.WriteString("srn:scp:sns:")
		sb.WriteString(s.cfg.Region)
		sb.WriteByte(':')
		sb.WriteString(account)
		sb.WriteByte(':')
		sb.WriteString(name)
		topics = append(topics, entity.Topic{TopicSrn: sb.String()})
	}
	return topics, nil
}

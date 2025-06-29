package service

import (
	"context"
	"nats/internal/context/traces"
	"nats/internal/entity"
	"nats/internal/repo"
	"nats/pkg/config"
	"strings"
)

type TopicService interface {
	CreateTopic(ctx context.Context, name, account string) (entity.Topic, error)
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

func (s *topicService) CreateTopic(ctx context.Context, name, account string) (entity.Topic, error) {
	_, err := s.natsRepo.CreateStream(ctx, name)
	topic := makeTopicSrn(s.cfg.Region, account, name)
	return topic, err
}

func (s *topicService) DeleteTopic(ctx context.Context, name string) error {
	return s.natsRepo.DeleteStream(ctx, name)
}

func (s *topicService) ListTopics(ctx context.Context, account string) ([]entity.Topic, error) {
	ctx, span := traces.StartSpan(ctx, "listTopics")
	defer span.End()

	namesCh, err := s.natsRepo.ListStreamNames(ctx)
	if err != nil {
		traces.RecordSpanError(ctx, span, "natsRepo.ListStreamNames error", err)
		return nil, err
	}

	var topics []entity.Topic
	for name := range namesCh {
		topics = append(topics, makeTopicSrn(s.cfg.Region, account, name))
	}
	return topics, nil
}

func makeTopicSrn(region, account, name string) entity.Topic {
	var sb strings.Builder
	sb.Grow(len("srn:scp:sns:::") + len(region) + len(account) + len(name))
	sb.WriteString("srn:scp:sns:")
	sb.WriteString(region)
	sb.WriteByte(':')
	sb.WriteString(account)
	sb.WriteByte(':')
	sb.WriteString(name)
	return entity.Topic{TopicSrn: sb.String()}
}

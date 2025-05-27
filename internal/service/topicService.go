// internal/service/topicService.go
package service

import (
	"context"
	"time"

	"nats/internal/context/traces"
	natsrepo "nats/internal/infra/nats"
	"nats/pkg/logger"

	"github.com/nats-io/nats.go"
)

func CreateTopic(ctx context.Context, name, subject string) error {
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

func DeleteTopic(ctx context.Context, name string) error {
	js := natsrepo.GetJetStream(ctx)
	return js.DeleteStream(name)
}

func ListTopics(ctx context.Context) ([]string, error) {
	ctx, span := traces.StartSpan(ctx, "listopics")
	defer span.End()

	logger.Info(ctx, "ListTopics New Span test")
	js := natsrepo.GetJetStream(ctx)

	var topics []string
	lister := js.StreamNames()
	for name := range lister {
		topics = append(topics, "srn:scp:sns:kr-cp-1:100000000000:"+name)
	}
	return topics, nil
}

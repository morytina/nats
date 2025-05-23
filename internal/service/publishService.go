// internal/service/publishService.go
package service

import (
	"context"
	"errors"
	"strconv"

	natsrepo "nats/internal/infra/nats"
)

func PublishMessage(ctx context.Context, topicName, message, subject string) (string, error) {
	if topicName == "" || message == "" {
		return "", errors.New("missing required fields")
	}

	if subject == "" {
		subject = topicName
	}

	js := natsrepo.GetJetStream(ctx)
	ack, err := js.Publish(subject, []byte(message))
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(ack.Sequence, 10), nil
}

func PublishAsyncMessage(ctx context.Context, topicName, message, subject string) (string, error) {
	if topicName == "" || message == "" {
		return "", errors.New("missing required fields")
	}

	if subject == "" {
		subject = topicName
	}

	js := natsrepo.GetJetStream(ctx)
	ack, err := js.PublishAsync(subject, []byte(message))
	if err != nil {
		return "", err
	}

	return ack.Msg().Reply, nil
}

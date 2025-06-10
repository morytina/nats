package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"nats/internal/context/logs"
	"nats/internal/infra/valkey"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type AckResult struct {
	Status   string `json:"status"`   // "PENDING", "ACK", "FAILED", "TIMEOUT"
	Sequence uint64 `json:"sequence"` // JetStream Sequence if ACK
}

type PublishService interface {
	PublishAsyncMessage(ctx context.Context, topicName, message, subject string) (string, error)
	CheckAckStatus(ctx context.Context, id string) (string, error)
}

type JetStreamClient interface {
	GetJetStream(ctx context.Context) nats.JetStreamContext
}

type publishService struct {
	jsClient   JetStreamClient
	dispatcher AckDispatcher
	timeout    time.Duration
}

func NewPublishService(jsClient JetStreamClient, dispatcher AckDispatcher, timeout time.Duration) PublishService {
	return &publishService{
		jsClient:   jsClient,
		dispatcher: dispatcher,
		timeout:    timeout,
	}
}

func (s *publishService) PublishAsyncMessage(ctx context.Context, topicName, message, subject string) (string, error) {
	logger := logs.GetLogger(ctx)
	logger.Debug("PublishAsyncMessage", logs.WithTraceFields(ctx)...)

	if topicName == "" || message == "" {
		return "", errors.New("missing required fields")
	}
	if subject == "" {
		subject = topicName
	}

	js := s.jsClient.GetJetStream(ctx)
	ackFuture, err := js.PublishAsync(subject, []byte(message))
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	_ = storeAckResult(ctx, id, AckResult{Status: "PENDING"})

	task := NewAckTask(ctx, id, ackFuture, s.timeout)
	s.dispatcher.Enqueue(task)

	return id, nil
}

func (s *publishService) CheckAckStatus(ctx context.Context, id string) (string, error) {
	jsonStr, err := valkey.GetClient().GetKey(ctx, id)
	if err != nil || jsonStr == "" {
		return "", errors.New("not found")
	}

	var result AckResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", err
	}

	switch result.Status {
	case "PENDING":
		return "PENDING", nil
	case "ACK":
		return "ACK " + strconv.FormatUint(result.Sequence, 10), nil
	case "FAILED":
		return "FAILED", nil
	default:
		return "", errors.New("unknown status")
	}
}

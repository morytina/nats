package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"nats/internal/context/logs"
	natsrepo "nats/internal/infra/nats"
	valkeyrepo "nats/internal/infra/valkey"

	"github.com/google/uuid"
)

type AckResult struct {
	Status   string `json:"status"`   // "PENDING", "ACK", "FAILED", "TIMEOUT"
	Sequence uint64 `json:"sequence"` // JetStream Sequence if ACK
}

type PublishService interface {
	PublishAsyncMessage(ctx context.Context, topicName, message, subject string) (string, error)
	CheckAckStatus(ctx context.Context, id string) (string, error)
}

type publishService struct {
	dispatcher AckDispatcher
	timeout    time.Duration
}

func NewPublishService(dispatcher AckDispatcher, timeout time.Duration) PublishService {
	return &publishService{
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

	js := natsrepo.GetJetStream(ctx)
	ackFuture, err := js.PublishAsync(subject, []byte(message))
	if err != nil {
		return "", err
	}

	id := uuid.NewString()

	// 초기 상태 저장
	_ = storeAckResult(ctx, id, AckResult{Status: "PENDING"})

	// AckTask 생성 및 큐로 전달
	task := NewAckTask(ctx, id, ackFuture, s.timeout)
	s.dispatcher.Enqueue(task)

	return id, nil
}

func (s *publishService) CheckAckStatus(ctx context.Context, id string) (string, error) {
	jsonStr, err := valkeyrepo.GetClient().GetKey(ctx, id)
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

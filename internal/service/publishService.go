package service

import (
	"context"
	"encoding/json"
	"errors"
	"nats/internal/context/logs"
	"nats/internal/infra/nats"
	"nats/internal/infra/valkey"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
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
	jsClient   nats.JetStreamPool
	dispatcher AckDispatcher
	timeout    time.Duration
}

func NewPublishService(jsClient nats.JetStreamPool, dispatcher AckDispatcher, timeout time.Duration) PublishService {
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

	// taskCtx is for goroutine context. So, make new context (without cancel, include span and logger)
	taskCtx := context.WithoutCancel(ctx)
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		taskCtx = trace.ContextWithSpanContext(taskCtx, spanCtx)
	}
	taskCtx = logs.WithLogger(taskCtx, logger)
	id := uuid.NewString()
	_ = storeAckResult(taskCtx, id, AckResult{Status: "PENDING"})

	task := NewAckTask(taskCtx, id, ackFuture, s.timeout)
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

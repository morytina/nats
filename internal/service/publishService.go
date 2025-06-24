package service

import (
	"context"
	"encoding/json"
	"errors"
	"nats/internal/context/logs"
	"nats/internal/entity"
	"nats/internal/repo"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/trace"
)

type PublishService interface {
	PublishAsyncMessage(ctx context.Context, topicName, message, subject string) (string, error)
	CheckAckStatus(ctx context.Context, id string) (string, error)
}

type publishService struct {
	dispatcher AckDispatcher
	timeout    time.Duration
	natsRepo   repo.NatsRepo
	valkeyRepo repo.ValkeyRepo
}

func NewPublishService(dispatcher AckDispatcher, timeout time.Duration, natsRepo repo.NatsRepo, valkeyRepo repo.ValkeyRepo) PublishService {
	return &publishService{
		dispatcher: dispatcher,
		timeout:    timeout,
		natsRepo:   natsRepo,
		valkeyRepo: valkeyRepo,
	}
}

// creates a new AckTask with its own timeout context.
func newAckTask(parentCtx context.Context, id string, future jetstream.PubAckFuture, timeout time.Duration) *entity.AckTask {
	return &entity.AckTask{
		ID:        id,
		Ctx:       parentCtx,
		AckFuture: future,
		TimeOut:   timeout,
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

	ackFuture, err := s.natsRepo.PublishAsyncMessage(ctx, message, subject)
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
	_ = s.valkeyRepo.StoreAckResult(taskCtx, id, entity.AckResult{Status: "PENDING"})

	task := newAckTask(taskCtx, id, ackFuture, s.timeout)
	s.dispatcher.Enqueue(task)

	return id, nil
}

func (s *publishService) CheckAckStatus(ctx context.Context, id string) (string, error) {
	jsonStr, err := s.valkeyRepo.GetAckStatus(ctx, id)

	if err != nil || jsonStr == "" {
		return "", errors.New("not found")
	}

	var result entity.AckResult
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

// internal/service/publishService.go
package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	natsrepo "nats/internal/infra/nats"

	"github.com/google/uuid"
)

type AckResult struct {
	Status   string // "PENDING", "ACK", "FAILED"
	Sequence uint64 // JetStream Sequence if ACK
}

// resultMap은 사용자가 확인할 결과를 저장
var resultMap = sync.Map{} // map[string]AckResult

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

	id := uuid.NewString()
	resultMap.Store(id, AckResult{
		Status:   "ACK",
		Sequence: ack.Sequence,
	})

	return id, nil
}

func PublishAsyncMessage(ctx context.Context, topicName, message, subject string) (string, error) {
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
	resultMap.Store(id, AckResult{Status: "PENDING"})

	go func() {
		select {
		case ack := <-ackFuture.Ok():
			if ack != nil {
				resultMap.Store(id, AckResult{
					Status:   "ACK",
					Sequence: ack.Sequence,
				})
			} else {
				resultMap.Store(id, AckResult{
					Status: "FAILED",
				})
			}
		case <-time.After(10 * time.Second):
			resultMap.Store(id, AckResult{
				Status: "FAILED",
			})
		}
	}()

	return id, nil
}

func CheckAckStatus(id string) (string, error) {
	value, ok := resultMap.Load(id)
	if !ok {
		return "", errors.New("not found")
	}

	result := value.(AckResult)

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

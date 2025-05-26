package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	natsrepo "nats/internal/infra/nats"
	valkeyrepo "nats/internal/infra/valkey"
	"nats/pkg/logger"

	"github.com/google/uuid"
)

type AckResult struct {
	Status   string `json:"status"`   // "PENDING", "ACK", "FAILED", "TIMEOUT"
	Sequence uint64 `json:"sequence"` // JetStream Sequence if ACK
}

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

	if ack != nil {
		storeAckResult(ctx, id, AckResult{
			Status:   "ACK",
			Sequence: ack.Sequence,
		})
	} else {
		storeAckResult(ctx, id, AckResult{Status: "FAILED"})
	}

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
	storeAckResult(ctx, id, AckResult{Status: "PENDING"})

	go func() {
		goCtx := context.Background()
		select {
		case ack := <-ackFuture.Ok():
			if ack != nil {
				storeAckResult(goCtx, id, AckResult{
					Status:   "ACK",
					Sequence: ack.Sequence,
				})
			} else {
				storeAckResult(goCtx, id, AckResult{Status: "FAILED"})
			}
		case <-time.After(10 * time.Second):
			storeAckResult(goCtx, id, AckResult{Status: "TIMEOUT"})
		}
	}()

	return id, nil
}

func CheckAckStatus(ctx context.Context, id string) (string, error) {
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

func storeAckResult(ctx context.Context, id string, result AckResult) error {
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	err = valkeyrepo.GetClient().SetKeyWithTTL(ctx, id, string(bytes), 30*time.Second)
	if err != nil {
		logger.Warn(ctx, "ACK 상태 저장 실패", "id", id, "error", err)
	}
	return err
}

package service

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
)

// AckTask represents an individual publish ack to be tracked.
type AckTask struct {
	ID        string
	Ctx       context.Context
	AckFuture nats.PubAckFuture
	TimeOut   time.Duration
}

// NewAckTask creates a new AckTask with its own timeout context.
func NewAckTask(parentCtx context.Context, id string, future nats.PubAckFuture, timeout time.Duration) *AckTask {
	return &AckTask{
		ID:        id,
		Ctx:       parentCtx,
		AckFuture: future,
		TimeOut:   timeout,
	}
}

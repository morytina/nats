package service

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// AckTask represents an individual publish ack to be tracked.
type AckTask struct {
	ID        string
	Ctx       context.Context
	AckFuture jetstream.PubAckFuture
	TimeOut   time.Duration
}

// NewAckTask creates a new AckTask with its own timeout context.
func NewAckTask(parentCtx context.Context, id string, future jetstream.PubAckFuture, timeout time.Duration) *AckTask {
	return &AckTask{
		ID:        id,
		Ctx:       parentCtx,
		AckFuture: future,
		TimeOut:   timeout,
	}
}

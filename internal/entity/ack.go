package entity

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type AckResult struct {
	Status   string `json:"status"`   // "PENDING", "ACK", "FAILED", "TIMEOUT"
	Sequence uint64 `json:"sequence"` // JetStream Sequence if ACK
}

// AckTask represents an individual publish ack to be tracked.
type AckTask struct {
	ID        string
	Ctx       context.Context
	AckFuture jetstream.PubAckFuture
	TimeOut   time.Duration
}

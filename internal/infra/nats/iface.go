package nats

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"
)

// JetStreamPool 인터페이스는 JetStreamContext를 반환하는 계약입니다.
type JetStreamPool interface {
	GetJetStream(ctx context.Context) jetstream.JetStream
}

var jetStreamClient JetStreamPool

func SetJetStreamClient(client JetStreamPool) {
	jetStreamClient = client
}

func GetJetStreamClient() JetStreamPool {
	return jetStreamClient
}

func GetJetStream(ctx context.Context) jetstream.JetStream {
	return jetStreamClient.GetJetStream(ctx)
}

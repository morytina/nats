package nats

import (
	"context"
	"fmt"
	"nats/pkg/glogger"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	ncPool    []*nats.Conn
	jsPool    []jetstream.JetStream
	nextJSIdx uint32
	pool      int
)

// defaultJetStreamClient는 실제 JetStream 연결 풀을 관리합니다.
type defaultJetStreamClient struct{}

func (c *defaultJetStreamClient) GetJetStream(ctx context.Context) jetstream.JetStream {
	for i := 0; i < pool; i++ {
		idx := int(atomic.AddUint32(&nextJSIdx, 1)) % pool
		nc := ncPool[idx]
		js := jsPool[idx]

		if nc == nil || nc.IsClosed() || !nc.IsConnected() {
			glogger.Warn(ctx, "JetStream 연결 문제", "index", idx)
			connName := fmt.Sprintf("SNS-API-Conn-%d", idx)
			opts := makeNATSOptions(ctx, connName)

			newNc, err := nats.Connect(nats.DefaultURL, opts...)
			if err != nil {
				glogger.Error(ctx, "재연결 실패", "index", idx, "error", err)
				continue
			}
			ncPool[idx] = newNc

			newJs, err := jetstream.New(newNc)
			if err != nil {
				glogger.Error(ctx, "JetStreamContext 재생성 실패", "index", idx, "error", err)
				continue
			}
			jsPool[idx] = newJs
			glogger.Info(ctx, "JetStream 재연결 성공", "index", idx)
			return newJs
		}
		return js
	}
	glogger.Error(ctx, "사용 가능한 JetStream 연결 없음")
	return jsPool[0]
}

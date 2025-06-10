package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"nats/internal/context/logs"
	"nats/internal/context/traces"
	valkeyrepo "nats/internal/infra/valkey"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// AckDispatcher defines the interface for processing async publish ACKs
type AckDispatcher interface {
	Start()
	Stop()
	Enqueue(task *AckTask)
}

type ackDispatcher struct {
	queue    chan *AckTask
	stopChan chan struct{}
	wg       sync.WaitGroup
	size     int
	worker   int
}

// NewAckDispatcher creates an AckDispatcher with the given queue size
func NewAckDispatcher(size, worker int) AckDispatcher {
	return &ackDispatcher{
		queue:    make(chan *AckTask, size),
		stopChan: make(chan struct{}),
		size:     size,
		worker:   worker,
	}
}

// Start launches the dispatcher loop
func (d *ackDispatcher) Start() {
	if d.worker == 0 {
		d.worker = 1
	}

	for i := 0; i < d.worker; i++ {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for {
				select {
				case task := <-d.queue:
					d.process(task)
				case <-d.stopChan:
					return
				}
			}
		}()
	}
}

// Stop signals all workers to exit and waits for them to finish
func (d *ackDispatcher) Stop() {
	close(d.stopChan)
	d.wg.Wait()
}

// Enqueue adds an AckTask to the queue for processing
func (d *ackDispatcher) Enqueue(task *AckTask) {
	d.queue <- task
}

// process handles a single AckTask and stores result in valkey
func (d *ackDispatcher) process(task *AckTask) {
	parentCtx := task.Ctx
	logger := logs.GetLogger(parentCtx)
	spanCtx := trace.SpanContextFromContext(parentCtx)

	var ctx context.Context
	if spanCtx.IsValid() {
		ctx = trace.ContextWithSpanContext(context.Background(), spanCtx)
	} else {
		ctx = parentCtx
	}

	ctx, span := traces.StartSpan(ctx, "ack.wait")
	defer span.End()

	select {
	case ack := <-task.AckFuture.Ok():
		if ack != nil {
			logger.Info("ACK received successfully", logs.WithTraceFields(ctx, zap.String("id", task.ID), zap.Uint64("seq", ack.Sequence))...)
			span.SetStatus(codes.Ok, "ACK received successfully")
			_ = storeAckResult(ctx, task.ID, AckResult{Status: "ACK", Sequence: ack.Sequence})
		} else {
			logger.Error("ACK reception failure", logs.WithTraceFields(ctx, zap.String("id", task.ID))...)
			span.SetStatus(codes.Error, "ACK reception failure")
			_ = storeAckResult(ctx, task.ID, AckResult{Status: "FAILED"})
		}
	case <-time.After(task.TimeOut):
		logger.Warn("ACK receive timeout", logs.WithTraceFields(ctx, zap.String("id", task.ID))...)
		span.SetStatus(codes.Error, "ACK receive timeout")
		_ = storeAckResult(ctx, task.ID, AckResult{Status: "TIMEOUT"})
	}
}

// storeAckResult persists the AckResult into valkey
func storeAckResult(ctx context.Context, id string, result AckResult) error {
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = valkeyrepo.GetClient().SetKeyWithTTL(ctx, id, string(bytes), 30*time.Second)
	if err != nil {
		logs.GetLogger(ctx).Warn("Failed to save ACK status", zap.String("id", id), zap.Error(err))
	}
	return err
}

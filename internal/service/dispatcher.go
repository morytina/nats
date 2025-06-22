package service

import (
	"nats/internal/context/logs"
	"nats/internal/context/traces"
	"nats/internal/entity"
	"nats/internal/repo"
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// AckDispatcher defines the interface for processing async publish ACKs
type AckDispatcher interface {
	Start()
	Stop()
	Enqueue(task *entity.AckTask)
}

type ackDispatcher struct {
	queue      chan *entity.AckTask
	stopChan   chan struct{}
	wg         sync.WaitGroup
	size       int
	worker     int
	valkeyRepo repo.ValkeyRepo
}

// NewAckDispatcher creates an AckDispatcher with the given queue size
func NewAckDispatcher(size, worker int, valkeyRepo repo.ValkeyRepo) AckDispatcher {
	return &ackDispatcher{
		queue:      make(chan *entity.AckTask, size),
		stopChan:   make(chan struct{}),
		size:       size,
		worker:     worker,
		valkeyRepo: valkeyRepo,
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
func (d *ackDispatcher) Enqueue(task *entity.AckTask) {
	d.queue <- task
}

// process handles a single AckTask and stores result in valkey
func (d *ackDispatcher) process(task *entity.AckTask) {
	ctx := task.Ctx
	logger := logs.GetLogger(ctx)

	ctx, span := traces.StartSpan(ctx, "ack.wait")
	defer span.End()

	select {
	case ack := <-task.AckFuture.Ok():
		if ack != nil {
			logger.Info("ACK received successfully", logs.WithTraceFields(ctx, zap.String("id", task.ID), zap.Uint64("seq", ack.Sequence))...)
			span.SetStatus(codes.Ok, "ACK received successfully")
			_ = d.valkeyRepo.StoreAckResult(ctx, task.ID, entity.AckResult{Status: "ACK", Sequence: ack.Sequence})
		} else {
			logger.Error("ACK reception failure", logs.WithTraceFields(ctx, zap.String("id", task.ID))...)
			span.SetStatus(codes.Error, "ACK reception failure")
			_ = d.valkeyRepo.StoreAckResult(ctx, task.ID, entity.AckResult{Status: "FAILED"})
		}
	case <-time.After(task.TimeOut):
		logger.Warn("ACK receive timeout", logs.WithTraceFields(ctx, zap.String("id", task.ID))...)
		span.SetStatus(codes.Error, "ACK receive timeout")
		_ = d.valkeyRepo.StoreAckResult(ctx, task.ID, entity.AckResult{Status: "TIMEOUT"})
	}
}

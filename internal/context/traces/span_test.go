package traces_test

import (
	"context"
	"errors"
	"testing"

	"nats/internal/context/logs"
	"nats/internal/context/traces"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func setupTestLogger() (*observer.ObservedLogs, *zap.Logger, context.Context) {
	core, observed := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	// Create context with logger injected
	ctx := logs.WithLogger(context.Background(), logger)
	return observed, logger, ctx
}

func TestStartSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	ctx := context.Background()
	ctx, span := traces.StartSpan(ctx, "test-span")
	assert.NotNil(t, span)
	assert.True(t, trace.SpanFromContext(ctx).SpanContext().IsValid())
	span.End()
}

func TestStartSpanWithAttrs(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	ctx := context.Background()
	attr := attribute.String("key", "value")

	ctx, span := traces.StartSpanWithAttrs(ctx, "test-span-attr", attr)
	defer span.End()

	assert.True(t, trace.SpanFromContext(ctx).SpanContext().IsValid())
}

func TestRecordSpanError(t *testing.T) {
	// Set up tracer
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Set up test logger and context
	observedLogs, _, ctx := setupTestLogger()

	// Start span
	ctx, span := traces.StartSpan(ctx, "record-span")
	defer span.End()

	// Trigger log
	err := errors.New("test error")
	traces.RecordSpanError(ctx, span, "failed to do something", err)

	// Verify log
	logs := observedLogs.FilterMessageSnippet("failed to do something").All()
	assert.Len(t, logs, 1)
	assert.Contains(t, logs[0].Message, "failed to do something")
	assert.Equal(t, "error", logs[0].Level.String())
}

func TestRecordSpanError_RecordsErrorInSpan(t *testing.T) {
	// 1. tracetest의 SpanRecorder 사용
	recorder := tracetest.NewSpanRecorder()

	// 2. TracerProvider에 span processor로 등록
	tp := sdktrace.NewTracerProvider()
	tp.RegisterSpanProcessor(recorder)
	otel.SetTracerProvider(tp)

	// 3. context + 로거 세팅
	_, _, ctx := setupTestLogger()
	ctx, span := traces.StartSpan(ctx, "span-with-error")

	err := errors.New("injected error")
	traces.RecordSpanError(ctx, span, "failure message", err)
	span.End()

	// 4. recorder에서 종료된 span 확인
	spans := recorder.Ended()
	assert.Len(t, spans, 1)

	s := spans[0]

	// 5. span 상태 확인
	assert.Equal(t, codes.Error, s.Status().Code)
	assert.Contains(t, s.Status().Description, "injected error")

	// 6. error 이벤트가 기록됐는지 확인
	var found bool
	for _, e := range s.Events() {
		if e.Name == "exception" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected span to contain exception event")
}

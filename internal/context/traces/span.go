package traces

import (
	"context"

	"nats/internal/context/logs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer("sns-api").Start(ctx, name)
}

func StartSpanWithAttrs(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer("sns-api")
	ctx, span := tracer.Start(ctx, name)
	span.SetAttributes(attrs...)
	return ctx, span
}

// RecordSpanError sets span status and records error with structured logging
func RecordSpanError(ctx context.Context, span trace.Span, msg string, err error) {
	if err == nil {
		return
	}
	span.SetStatus(codes.Error, err.Error())
	span.RecordError(err)
	logs.GetLogger(ctx).Error(msg, logs.WithTraceFields(ctx, zap.Error(err))...)
}

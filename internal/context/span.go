package context

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

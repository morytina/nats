package logs

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var loggerKey = ctxKey{}

// WithLogger stores a zap.Logger in the context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger retrieves a zap.Logger from the context, or returns zap.NewNop() if not found
func GetLogger(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

// WithFields returns a new context with additional fields applied to the logger
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	logger := GetLogger(ctx).With(fields...)
	return WithLogger(ctx, logger)
}

// NewLogger creates a base zap.Logger with optional initial fields
func NewLogger(levelStr string, fields ...zap.Field) (*zap.Logger, error) {
	level := parseLevel(levelStr)

	cfg := zap.Config{
		Encoding:         "console",
		Level:            zap.NewAtomicLevelAt(level),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	baseLogger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return baseLogger.With(fields...), nil
}

// parseLevel converts a string log level into zapcore.Level
func parseLevel(str string) zapcore.Level {
	switch strings.ToLower(str) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// InjectTrace exposes trace_id and span_id fields for manual injection
func injectTrace(ctx context.Context) []zap.Field {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}
	return []zap.Field{
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	}
}

// WithTraceFields appends trace_id and span_id to user-provided fields
func WithTraceFields(ctx context.Context, fields ...zap.Field) []zap.Field {
	return append(fields, injectTrace(ctx)...)
}

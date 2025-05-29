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

// parseLevel converts a string log level into a zapcore.Level
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

// injectTracing returns trace_id and span_id fields if available
func injectTracing(ctx context.Context) []zap.Field {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}
	return []zap.Field{
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	}
}

// Logging helpers with trace injection
func DebugWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(zap.AddCallerSkip(1)).Debug(msg, append(fields, injectTracing(ctx)...)...)
}

func InfoWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(zap.AddCallerSkip(1)).Info(msg, append(fields, injectTracing(ctx)...)...)
}

func WarnWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(zap.AddCallerSkip(1)).Warn(msg, append(fields, injectTracing(ctx)...)...)
}

func ErrorWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(zap.AddCallerSkip(1)).Error(msg, append(fields, injectTracing(ctx)...)...)
}

func FatalWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(zap.AddCallerSkip(1)).Fatal(msg, append(fields, injectTracing(ctx)...)...)
}

func PanicWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	GetLogger(ctx).WithOptions(zap.AddCallerSkip(1)).Panic(msg, append(fields, injectTracing(ctx)...)...)
}

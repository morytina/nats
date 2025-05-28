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

func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func GetLogger(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return zap.NewNop().Sugar()
}

// 새 필드 추가된 logger로 context 갱신
func WithFields(ctx context.Context, fields ...any) context.Context {
	logger := GetLogger(ctx).With(fields...)
	return WithLogger(ctx, logger)
}

func NewLogger(levelStr string, fields ...zap.Field) (*zap.SugaredLogger, error) {
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
	coreLogger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return coreLogger.With(fields...).Sugar(), nil
}

// parseLevel converts string to zapcore.Level
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

// injectTracing appends trace_id and span_id to keyvals if available
func injectTracing(ctx context.Context, keyvals []interface{}) []interface{} {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		keyvals = append(keyvals,
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		)
	}
	return keyvals
}

func DebugWithTrace(ctx context.Context, msg string, keyvals ...interface{}) {
	GetLogger(ctx).Debugw(msg, injectTracing(ctx, keyvals)...)
}

func InfoWithTrace(ctx context.Context, msg string, keyvals ...interface{}) {
	GetLogger(ctx).Infow(msg, injectTracing(ctx, keyvals)...)
}

func WarnWithTrace(ctx context.Context, msg string, keyvals ...interface{}) {
	GetLogger(ctx).Warnw(msg, injectTracing(ctx, keyvals)...)
}

func ErrorWithTrace(ctx context.Context, msg string, keyvals ...interface{}) {
	GetLogger(ctx).Errorw(msg, injectTracing(ctx, keyvals)...)
}

func FatalWithTrace(ctx context.Context, msg string, keyvals ...interface{}) {
	GetLogger(ctx).Fatalw(msg, injectTracing(ctx, keyvals)...)
}

func PanicWithTrace(ctx context.Context, msg string, keyvals ...interface{}) {
	GetLogger(ctx).Panicw(msg, injectTracing(ctx, keyvals)...)
}

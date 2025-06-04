package glogger

import (
	"context"
	"strings"

	"nats/pkg/config"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

type LogLevel zapcore.Level

const (
	LevelDebug LogLevel = LogLevel(zapcore.DebugLevel)
	LevelInfo  LogLevel = LogLevel(zapcore.InfoLevel)
	LevelWarn  LogLevel = LogLevel(zapcore.WarnLevel)
	LevelError LogLevel = LogLevel(zapcore.ErrorLevel)
	LevelFatal LogLevel = LogLevel(zapcore.FatalLevel)
	LevelPanic LogLevel = LogLevel(zapcore.PanicLevel)
)

type CtxKey string

const RequestIDKey CtxKey = "request_id"

func GlobalLogger(cfg *config.Config) {
	initWithLevel(parseLevel(cfg.Log.Level))
}

func parseLevel(str string) LogLevel {
	switch strings.ToLower(str) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	case "panic":
		return LevelPanic
	default:
		return LevelInfo
	}
}

func initWithLevel(level LogLevel) {
	cfg := zap.Config{
		Encoding:         "console", // 운영환경 json 변경 고려
		Level:            zap.NewAtomicLevelAt(zapcore.Level(level)),
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
		panic(err)
	}

	log = baseLogger.
		WithOptions(zap.AddCallerSkip(1)).
		Sugar()
}

// Context-required logging methods
func Debug(ctx context.Context, msg string, kv ...interface{}) {
	log.Debugw(msg, convertToFields(ctx, kv...)...)
}
func Info(ctx context.Context, msg string, kv ...interface{}) {
	log.Infow(msg, convertToFields(ctx, kv...)...)
}
func Warn(ctx context.Context, msg string, kv ...interface{}) {
	log.Warnw(msg, convertToFields(ctx, kv...)...)
}
func Error(ctx context.Context, msg string, kv ...interface{}) {
	log.Errorw(msg, convertToFields(ctx, kv...)...)
}
func Fatal(ctx context.Context, msg string, kv ...interface{}) {
	log.Fatalw(msg, convertToFields(ctx, kv...)...)
}

func convertToFields(ctx context.Context, kv ...interface{}) []interface{} {
	fields := make([]interface{}, 0, len(kv)+4)
	fields = append(fields, kv...)

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		fields = append(fields,
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		)
	}

	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		fields = append(fields, "request_id", reqID)
	}
	return fields
}

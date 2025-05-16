package logger

import (
	"strings"

	"nats/internal/config"

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

// Init initializes the global logger using configs/config.yaml.
// The caller field shows the location of logger.Info(), etc.
func Init() {
	const configPath = "configs/config.yaml" // ← 경로 변경됨

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		panic("logger config load error: " + err.Error())
	}

	initWithLevel(parseLevel(cfg.Level))
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
		WithOptions(zap.AddCallerSkip(1)). // caller 위치를 외부 호출자로 보정
		Sugar()
}

// Global logging methods
func Debug(msg string, kv ...interface{}) { log.Debugw(msg, kv...) }
func Info(msg string, kv ...interface{})  { log.Infow(msg, kv...) }
func Warn(msg string, kv ...interface{})  { log.Warnw(msg, kv...) }
func Error(msg string, kv ...interface{}) { log.Errorw(msg, kv...) }
func Fatal(msg string, kv ...interface{}) { log.Fatalw(msg, kv...) }

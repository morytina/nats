package glogger

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerWithContextFields(t *testing.T) {
	// 기존 전역 로거 백업
	originalLog := log

	// 테스트용 observer logger 생성
	core, recorded := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core).Sugar()

	// 전역 log 변수 덮어쓰기
	log = testLogger

	// 테스트 종료 시 복원
	defer func() {
		log = originalLog
	}()

	// context 구성
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-req-id")
	tr := otel.Tracer("test")
	ctx, span := tr.Start(ctx, "test-span")
	defer span.End()

	// 로깅 함수 호출
	Info(ctx, "test message", "key1", "val1")

	// 로그 결과 검증
	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["key1"] != "val1" {
		t.Errorf("expected key1=val1, got %v", fields["key1"])
	}
	if fields["request_id"] != "test-req-id" {
		t.Errorf("expected request_id=test-req-id, got %v", fields["request_id"])
	}
	if fields["trace_id"] == "" {
		t.Errorf("expected trace_id to be present, got empty")
	}
	if fields["span_id"] == "" {
		t.Errorf("expected span_id to be present, got empty")
	}
}

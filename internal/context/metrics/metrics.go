// internal/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// API 호출 수 측정
	ApiCallCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "action_handler_calls_total",
			Help: "Total number of API calls by action",
		},
		[]string{"action", "status"},
	)

	// NATS 연결 상태 메트릭
	NatsReconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_reconnect_total",
			Help: "총 NATS 재연결 횟수",
		},
		[]string{"conn"},
	)
	NatsDisconnects = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_disconnect_total",
			Help: "총 NATS 연결 실패 횟수",
		},
		[]string{"conn"},
	)

	// Valkey 연결 상태 메트릭
	ValkeyReconnects = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "valkey_reconnect_total",
			Help: "총 Valkey 재연결 횟수",
		},
	)
	ValkeyFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "valkey_connection_failures_total",
			Help: "Valkey 연결 실패 횟수",
		},
	)
)

func StartMetrics() {
	prometheus.MustRegister(ApiCallCounter)
	prometheus.MustRegister(NatsReconnects)
	prometheus.MustRegister(NatsDisconnects)
	prometheus.MustRegister(ValkeyReconnects)
	prometheus.MustRegister(ValkeyFailures)
}

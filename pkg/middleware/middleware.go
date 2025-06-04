package middleware

import (
	"context"
	"nats/pkg/glogger"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// AttachMiddlewares sets up common middlewares to the Echo instance
func AttachMiddlewares(e *echo.Echo) {
	// 기본 로깅, 복구, Request ID
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// Request ID를 context에 추가
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			ctx = contextWithRequestID(ctx, requestID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	// OpenTelemetry trace context 추출
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request().Header))
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	// OpenTelemetry 핸들러 Wrapping
	e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "EchoRequest")
	}))
}

// context에 request_id 저장
func contextWithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, glogger.RequestIDKey, reqID)
}

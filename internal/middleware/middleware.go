package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	"nats/internal/context/logs"
)

// AttachMiddlewares sets up core middlewares
func AttachMiddlewares(e *echo.Echo, logger *zap.Logger) {
	// Wrap with OpenTelemetry
	e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "EchoRequest")
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Validator = NewCustomValidator() // validator.go

	// Inject request_id, trace_id, span_id into context + logger
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			// Get or generate request_id
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = uuid.NewString()
			}
			// Inject logger into context first so fields are attached to the correct logger
			ctx = logs.WithLogger(ctx, logger)
			ctx = logs.WithFields(ctx, zap.String("request_id", requestID))

			// 만약 otelhttp.NewHandler 에서 trace header 받아온다면 아래 코드는 완전 삭제.
			// ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	})
}

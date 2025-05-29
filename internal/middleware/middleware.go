package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"

	"nats/internal/context/logs"
)

// AttachMiddlewares sets up core middlewares
func AttachMiddlewares(e *echo.Echo, baseLogger *zap.Logger) {
	// OpenTelemetry HTTP trace wrapper
	e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "EchoRequest")
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// Inject trace_id, span_id, request_id into context + logger
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			// Extract trace context from incoming request headers
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

			// Get or generate request ID
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = uuid.NewString()
			}

			// Build enriched logger
			logger := baseLogger.With(
				zap.String("request_id", requestID),
			)

			// Store logger in context
			ctx = logs.WithLogger(ctx, logger)
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	})
}

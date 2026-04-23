package middleware

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

const requestIDMaxLen = 64

// validRequestID matches strings containing only alphanumeric characters and hyphens.
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

const (
	// RequestIDHeader é o header usado para o Request ID
	RequestIDHeader = "Request-ID"
	// legacyRequestIDHeader is accepted as input for backward compatibility
	// with clients still sending the X-prefixed variant. Response continues to
	// use RequestIDHeader only.
	legacyRequestIDHeader = "X-Request-ID"
)

// Logger retorna um middleware de logging estruturado
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Gerar ou usar Request ID existente (sanitized).
		// Fall back to the legacy X-Request-ID header so clients that have not
		// migrated yet still get trace-continuity.
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = c.GetHeader(legacyRequestIDHeader)
		}
		if requestID == "" || len(requestID) > requestIDMaxLen || !validRequestID.MatchString(requestID) {
			requestID = uuid.New().String()
		}
		c.Header(RequestIDHeader, requestID)

		// Extrair Trace ID se disponível
		traceID := ""
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().HasTraceID() {
			traceID = span.SpanContext().TraceID().String()
		}

		// Inject LogContext into request context for downstream use
		lc := logutil.LogContext{
			RequestID: requestID,
			TraceID:   traceID,
			Step:      logutil.StepMiddleware,
		}
		ctx := logutil.Inject(c.Request.Context(), lc)
		c.Request = c.Request.WithContext(ctx)

		// Log de entrada
		logutil.LogInfo(ctx, "request started",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"client_ip", c.ClientIP(),
		)

		// Processar request
		c.Next()

		// Log de saída
		duration := time.Since(start)
		status := c.Writer.Status()

		fields := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
		}

		switch {
		case status >= 500:
			logutil.LogError(ctx, "request completed", fields...)
		case status >= 400:
			logutil.LogWarn(ctx, "request completed", fields...)
		default:
			logutil.LogInfo(ctx, "request completed", fields...)
		}
	}
}

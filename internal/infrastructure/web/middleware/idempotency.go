package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/DenysonJ/financial-wallet/pkg/httputil"
	"github.com/DenysonJ/financial-wallet/pkg/idempotency"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
)

const (
	// IdempotencyKeyHeader is the header used for idempotency.
	IdempotencyKeyHeader = "Idempotency-Key"

	// idempotencyKeyPrefix is the prefix used in Redis keys.
	idempotencyKeyPrefix = "idempotency:"
)

// RequireIdempotencyKey returns a middleware that rejects requests missing the
// Idempotency-Key header with HTTP 400. Intended for endpoints where duplicate
// retries could cause data corruption (e.g. financial batch imports). It must
// be composed AFTER the global Idempotency middleware so replays can still be
// served. If store is nil (dev/test without Redis), the middleware is a no-op
// to match the graceful-degradation pattern used elsewhere.
func RequireIdempotencyKey(store idempotency.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.Next()
			return
		}
		if c.GetHeader(IdempotencyKeyHeader) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, httputil.ErrorResponse{
				Errors: httputil.ErrorDetail{Message: "Idempotency-Key header is required for this endpoint"},
			})
			return
		}
		c.Next()
	}
}

// Idempotency returns a middleware that ensures idempotency for POST requests.
// The Idempotency-Key header is optional: if absent, the request is processed normally.
// If Redis is unavailable, the middleware operates in fail-open mode (degrades gracefully).
func Idempotency(store idempotency.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Only applies to POST requests
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		// 2. Header is optional — if absent, process normally
		key := c.GetHeader(IdempotencyKeyHeader)
		if key == "" {
			c.Next()
			return
		}

		// 3. Read and buffer body for fingerprint
		reqBody, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil {
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		fingerprint := bodyFingerprint(reqBody)

		// 4. Build Redis key with optional service namespace
		serviceName := c.GetHeader("Service-Name")
		fullKey := buildIdempotencyKey(serviceName, key)

		ctx := c.Request.Context()

		// 5. Attempt to acquire lock
		acquired, lockErr := store.Lock(ctx, fullKey, fingerprint)
		if lockErr != nil {
			// Redis unavailable -> fail-open
			logutil.LogWarn(ctx, "idempotency store unavailable, proceeding without",
				"error", lockErr.Error(), "idempotency_key", key)
			c.Next()
			return
		}

		if !acquired {
			// Key already exists: check state
			entry, getErr := store.Get(ctx, fullKey)
			if getErr != nil {
				// Error fetching -> fail-open
				logutil.LogWarn(ctx, "idempotency store get failed, proceeding without",
					"error", getErr.Error(), "idempotency_key", key)
				c.Next()
				return
			}

			if entry == nil {
				// Key existed but expired between Lock and Get (rare race condition)
				c.Next()
				return
			}

			if entry.Status == idempotency.StatusProcessing {
				// Previous request still in progress -> 409 Conflict
				c.AbortWithStatusJSON(http.StatusConflict, httputil.ErrorResponse{
					Errors: httputil.ErrorDetail{Message: "A request with this Idempotency-Key is already being processed"},
				})
				return
			}

			// COMPLETED -> verify fingerprint before replay
			if entry.Fingerprint != "" && fingerprint != entry.Fingerprint {
				logutil.LogWarn(ctx, "idempotency key reused with different body",
					"idempotency_key", key)
				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, httputil.ErrorResponse{
					Errors: httputil.ErrorDetail{Message: "Idempotency-Key already used with a different request body"},
				})
				return
			}

			// Replay stored response
			logutil.LogInfo(ctx, "idempotency replay",
				"idempotency_key", key, "status_code", entry.StatusCode)
			c.Data(entry.StatusCode, "application/json; charset=utf-8", entry.Body)
			c.Abort()
			return
		}

		// 6. First request with this key — capture response
		rw := &idempotencyResponseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = rw

		// 7. Execute handler
		c.Next()

		// 8. Store or release based on status code
		statusCode := rw.Status()
		if shouldStoreResponse(statusCode) {
			completeErr := store.Complete(ctx, fullKey, &idempotency.Entry{
				StatusCode:  statusCode,
				Body:        rw.body.Bytes(),
				Fingerprint: fingerprint,
			})
			if completeErr != nil {
				logutil.LogWarn(ctx, "failed to store idempotency response",
					"error", completeErr.Error(), "idempotency_key", key)
			}
		} else {
			// 5xx error -> unlock to allow retry
			unlockErr := store.Unlock(ctx, fullKey)
			if unlockErr != nil {
				logutil.LogWarn(ctx, "failed to unlock idempotency key",
					"error", unlockErr.Error(), "idempotency_key", key)
			}
		}
	}
}

// buildIdempotencyKey builds the Redis key with namespace.
// Format: idempotency:{service-name}:{key} or idempotency:{key}
func buildIdempotencyKey(serviceName, key string) string {
	if serviceName != "" {
		return idempotencyKeyPrefix + serviceName + ":" + key
	}
	return idempotencyKeyPrefix + key
}

// shouldStoreResponse determines whether the response should be stored for replay.
// 2xx and 4xx are deterministic and should be stored.
// 5xx are transient and should allow retry.
func shouldStoreResponse(statusCode int) bool {
	return statusCode >= 200 && statusCode < 500
}

// idempotencyResponseWriter wraps gin.ResponseWriter to capture the response body.
type idempotencyResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *idempotencyResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// bodyFingerprint calculates the SHA-256 of the request body to detect reuse
// of Idempotency-Key with a different body.
func bodyFingerprint(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds the configuration for the rate limit middleware.
type RateLimitConfig struct {
	Store  ratelimit.Store
	Limit  int
	Window time.Duration
}

// RateLimit returns a middleware that enforces rate limiting per IP+method+route.
//
// Behavior:
//   - Builds key from client IP + HTTP method + matched route
//   - Checks rate limit via Store.Allow (Redis-backed)
//   - Sets RateLimit-Limit, RateLimit-Remaining, RateLimit-Reset headers
//   - Returns 429 Too Many Requests if limit exceeded
//   - Fail-open: if store returns error, logs warning and allows request
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		key := fmt.Sprintf("ratelimit:%s:%s:%s", clientIP, c.Request.Method, route)

		result, allowErr := cfg.Store.Allow(c.Request.Context(), key, cfg.Limit, cfg.Window)
		if allowErr != nil {
			logutil.LogWarn(c.Request.Context(), "rate limit store unavailable, proceeding without",
				"error", allowErr.Error(), "client_ip", clientIP)
			c.Next()
			return
		}

		// Set rate limit headers on every response
		c.Header("RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
		c.Header("RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		c.Header("RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		if !result.Allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(result.ResetAt).Seconds())))
			httpgin.SendError(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

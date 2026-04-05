package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DenysonJ/financial-wallet/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// --- Mock Store ---

type mockRateLimitStore struct {
	results []*ratelimit.Result // sequential results to return
	callIdx int
	err     error
}

func (m *mockRateLimitStore) Allow(_ context.Context, _ string, limit int, _ time.Duration) (*ratelimit.Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.callIdx < len(m.results) {
		r := m.results[m.callIdx]
		m.callIdx++
		return r, nil
	}
	// Default: allowed
	return &ratelimit.Result{
		Allowed:   true,
		Limit:     limit,
		Remaining: limit - 1,
		ResetAt:   time.Now().Add(1 * time.Minute),
	}, nil
}

func setupRateLimitRouter(store ratelimit.Store, limit int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(RateLimitConfig{
		Store:  store,
		Limit:  limit,
		Window: 1 * time.Minute,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestRateLimit_AllowedRequest(t *testing.T) {
	store := &mockRateLimitStore{
		results: []*ratelimit.Result{
			{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(1 * time.Minute)},
		},
	}
	r := setupRateLimitRouter(store, 100)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "100", w.Header().Get("RateLimit-Limit"))
	assert.Equal(t, "99", w.Header().Get("RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("RateLimit-Reset"))
}

func TestRateLimit_BlockedRequest(t *testing.T) {
	store := &mockRateLimitStore{
		results: []*ratelimit.Result{
			{Allowed: false, Limit: 100, Remaining: 0, ResetAt: time.Now().Add(30 * time.Second)},
		},
	}
	r := setupRateLimitRouter(store, 100)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "100", w.Header().Get("RateLimit-Limit"))
	assert.Equal(t, "0", w.Header().Get("RateLimit-Remaining"))
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
}

func TestRateLimit_HeadersPresentOn429(t *testing.T) {
	resetAt := time.Now().Add(45 * time.Second)
	store := &mockRateLimitStore{
		results: []*ratelimit.Result{
			{Allowed: false, Limit: 10, Remaining: 0, ResetAt: resetAt},
		},
	}
	r := setupRateLimitRouter(store, 10)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "10", w.Header().Get("RateLimit-Limit"))
	assert.Equal(t, "0", w.Header().Get("RateLimit-Remaining"))
	// Reset header should be a Unix timestamp
	assert.NotEmpty(t, w.Header().Get("RateLimit-Reset"))
}

func TestRateLimit_FailOpen_StoreError(t *testing.T) {
	store := &mockRateLimitStore{
		err: errors.New("redis connection refused"),
	}
	r := setupRateLimitRouter(store, 100)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should pass through (fail-open)
	assert.Equal(t, http.StatusOK, w.Code)
	// No rate limit headers when store fails
	assert.Empty(t, w.Header().Get("RateLimit-Limit"))
}

func TestRateLimit_MultipleRequests_CountsDown(t *testing.T) {
	store := &mockRateLimitStore{
		results: []*ratelimit.Result{
			{Allowed: true, Limit: 3, Remaining: 2, ResetAt: time.Now().Add(1 * time.Minute)},
			{Allowed: true, Limit: 3, Remaining: 1, ResetAt: time.Now().Add(1 * time.Minute)},
			{Allowed: true, Limit: 3, Remaining: 0, ResetAt: time.Now().Add(1 * time.Minute)},
			{Allowed: false, Limit: 3, Remaining: 0, ResetAt: time.Now().Add(1 * time.Minute)},
		},
	}
	r := setupRateLimitRouter(store, 3)

	// First 3 requests should pass
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 4th request should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

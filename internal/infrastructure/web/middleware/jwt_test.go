package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DenysonJ/financial-wallet/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupJWTTestRouter(jwtService *jwt.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(JWTAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestJWTAuth_ValidAccessToken(t *testing.T) {
	svc := jwt.NewService("test-secret-32chars-long!!", 15*time.Minute, 7*24*time.Hour)
	r := setupJWTTestRouter(svc)

	token, _ := svc.GenerateAccessToken("user-123")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	svc := jwt.NewService("test-secret-32chars-long!!", 15*time.Minute, 7*24*time.Hour)
	r := setupJWTTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	svc := jwt.NewService("test-secret-32chars-long!!", 15*time.Minute, 7*24*time.Hour)
	r := setupJWTTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	svc := jwt.NewService("test-secret-32chars-long!!", 1*time.Millisecond, 7*24*time.Hour)
	r := setupJWTTestRouter(svc)

	token, _ := svc.GenerateAccessToken("user-123")
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_RefreshTokenRejected(t *testing.T) {
	svc := jwt.NewService("test-secret-32chars-long!!", 15*time.Minute, 7*24*time.Hour)
	r := setupJWTTestRouter(svc)

	// Generate refresh token (should be rejected by middleware)
	token, _ := svc.GenerateRefreshToken("user-123")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_MalformedHeader(t *testing.T) {
	svc := jwt.NewService("test-secret-32chars-long!!", 15*time.Minute, 7*24*time.Hour)
	r := setupJWTTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "NotBearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

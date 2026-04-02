package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraauth "github.com/DenysonJ/financial-wallet/internal/infrastructure/auth"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupJWTTestRouter(tokenValidator interfaces.TokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(JWTAuth(tokenValidator))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func newTestTokenAdapter() (*jwt.Service, interfaces.TokenService) {
	svc := jwt.NewService("test-secret-32chars-long!!", 15*time.Minute, 7*24*time.Hour)
	return svc, infraauth.NewJWTTokenAdapter(svc)
}

func TestJWTAuth_ValidAccessToken(t *testing.T) {
	svc, adapter := newTestTokenAdapter()
	r := setupJWTTestRouter(adapter)

	token, _ := svc.GenerateAccessToken("user-123")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	_, adapter := newTestTokenAdapter()
	r := setupJWTTestRouter(adapter)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	_, adapter := newTestTokenAdapter()
	r := setupJWTTestRouter(adapter)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	expiredSvc := jwt.NewService("test-secret-32chars-long!!", 1*time.Millisecond, 7*24*time.Hour)
	expiredAdapter := infraauth.NewJWTTokenAdapter(expiredSvc)
	r := setupJWTTestRouter(expiredAdapter)

	token, _ := expiredSvc.GenerateAccessToken("user-123")
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_RefreshTokenRejected(t *testing.T) {
	svc, adapter := newTestTokenAdapter()
	r := setupJWTTestRouter(adapter)

	// Generate refresh token (should be rejected by middleware)
	token, _ := svc.GenerateRefreshToken("user-123")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_MalformedHeader(t *testing.T) {
	_, adapter := newTestTokenAdapter()
	r := setupJWTTestRouter(adapter)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "NotBearer token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

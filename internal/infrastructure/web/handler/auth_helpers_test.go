package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c
}

func TestIsServiceKeyRequest(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *gin.Context)
		expected bool
	}{
		{
			name:     "returns true when service key is set",
			setup:    func(c *gin.Context) { c.Set(middleware.ContextKeyServiceKey, "myservice") },
			expected: true,
		},
		{
			name:     "returns false when service key is not set",
			setup:    func(c *gin.Context) {},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext()
			tt.setup(c)
			assert.Equal(t, tt.expected, isServiceKeyRequest(c))
		})
	}
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *gin.Context)
		expected bool
	}{
		{
			name:     "returns true when user has admin role",
			setup:    func(c *gin.Context) { c.Set("user_roles", []string{"admin"}) },
			expected: true,
		},
		{
			name:     "returns true when user has admin among multiple roles",
			setup:    func(c *gin.Context) { c.Set("user_roles", []string{"user", "admin", "viewer"}) },
			expected: true,
		},
		{
			name:     "returns false when user has no admin role",
			setup:    func(c *gin.Context) { c.Set("user_roles", []string{"user", "viewer"}) },
			expected: false,
		},
		{
			name:     "returns false when user_roles not set",
			setup:    func(c *gin.Context) {},
			expected: false,
		},
		{
			name:     "returns false when user_roles is wrong type",
			setup:    func(c *gin.Context) { c.Set("user_roles", "admin") },
			expected: false,
		},
		{
			name:     "returns false when user_roles is empty slice",
			setup:    func(c *gin.Context) { c.Set("user_roles", []string{}) },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext()
			tt.setup(c)
			assert.Equal(t, tt.expected, isAdmin(c))
		})
	}
}

func TestIsAdminOrOwner(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(c *gin.Context)
		resourceUserID string
		expected       bool
	}{
		{
			name:           "returns true for service key request",
			setup:          func(c *gin.Context) { c.Set(middleware.ContextKeyServiceKey, "myservice") },
			resourceUserID: "user-999",
			expected:       true,
		},
		{
			name: "returns true when JWT user matches resource owner",
			setup: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "user-123")
			},
			resourceUserID: "user-123",
			expected:       true,
		},
		{
			name: "returns true when user is admin",
			setup: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "user-456")
				c.Set("user_roles", []string{"admin"})
			},
			resourceUserID: "user-123",
			expected:       true,
		},
		{
			name: "returns false when JWT user does not match and is not admin",
			setup: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "user-456")
				c.Set("user_roles", []string{"viewer"})
			},
			resourceUserID: "user-123",
			expected:       false,
		},
		{
			name:           "returns false when no auth context set",
			setup:          func(c *gin.Context) {},
			resourceUserID: "user-123",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext()
			tt.setup(c)
			assert.Equal(t, tt.expected, isAdminOrOwner(c, tt.resourceUserID))
		})
	}
}

func TestGetRequiredJWTUserID(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(c *gin.Context)
		expectedID string
		expectedOK bool
	}{
		{
			name:       "returns user ID when set",
			setup:      func(c *gin.Context) { c.Set(middleware.ContextKeyUserID, "user-123") },
			expectedID: "user-123",
			expectedOK: true,
		},
		{
			name:       "returns false when not set",
			setup:      func(c *gin.Context) {},
			expectedID: "",
			expectedOK: false,
		},
		{
			name:       "returns false when set to empty string",
			setup:      func(c *gin.Context) { c.Set(middleware.ContextKeyUserID, "") },
			expectedID: "",
			expectedOK: false,
		},
		{
			name:       "returns false when set to non-string type",
			setup:      func(c *gin.Context) { c.Set(middleware.ContextKeyUserID, 123) },
			expectedID: "",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext()
			tt.setup(c)
			id, ok := getRequiredJWTUserID(c)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestOwnershipUserID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *gin.Context)
		expected string
	}{
		{
			name: "returns empty for service key request (skip ownership)",
			setup: func(c *gin.Context) {
				c.Set(middleware.ContextKeyServiceKey, "myservice")
				c.Set(middleware.ContextKeyUserID, "user-123")
			},
			expected: "",
		},
		{
			name: "returns empty for admin user (skip ownership)",
			setup: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "user-123")
				c.Set("user_roles", []string{"admin"})
			},
			expected: "",
		},
		{
			name: "returns user ID for regular user",
			setup: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "user-123")
				c.Set("user_roles", []string{"viewer"})
			},
			expected: "user-123",
		},
		{
			name:     "returns empty when no auth context",
			setup:    func(c *gin.Context) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext()
			tt.setup(c)
			assert.Equal(t, tt.expected, ownershipUserID(c))
		})
	}
}

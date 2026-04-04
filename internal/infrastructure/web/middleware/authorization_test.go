package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/mocks/middlewaremock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(c *gin.Context)
		setupLoader    func(loader *middlewaremock.MockPermissionLoader)
		permission     string
		wantStatus     int
		wantPermStored bool
	}{
		{
			name:         "no user_id in context returns 401",
			setupContext: func(_ *gin.Context) {},
			setupLoader:  func(_ *middlewaremock.MockPermissionLoader) {},
			permission:   "user:read",
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name: "empty user_id returns 401",
			setupContext: func(c *gin.Context) {
				c.Set(ContextKeyUserID, "")
			},
			setupLoader: func(_ *middlewaremock.MockPermissionLoader) {},
			permission:  "user:read",
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name: "loader error returns 500",
			setupContext: func(c *gin.Context) {
				c.Set(ContextKeyUserID, "user-123")
			},
			setupLoader: func(loader *middlewaremock.MockPermissionLoader) {
				loader.On("GetPermissions", mock.Anything, "user-123").
					Return(nil, errors.New("db connection failed"))
			},
			permission: "user:read",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "missing permission returns 403",
			setupContext: func(c *gin.Context) {
				c.Set(ContextKeyUserID, "user-123")
			},
			setupLoader: func(loader *middlewaremock.MockPermissionLoader) {
				loader.On("GetPermissions", mock.Anything, "user-123").
					Return([]string{"user:read"}, nil)
			},
			permission: "user:delete",
			wantStatus: http.StatusForbidden,
		},
		{
			name: "has permission passes through",
			setupContext: func(c *gin.Context) {
				c.Set(ContextKeyUserID, "user-123")
			},
			setupLoader: func(loader *middlewaremock.MockPermissionLoader) {
				loader.On("GetPermissions", mock.Anything, "user-123").
					Return([]string{"user:read", "user:write"}, nil)
				loader.On("GetRoles", mock.Anything, "user-123").
					Return([]string{"admin"}, nil)
			},
			permission:     "user:read",
			wantStatus:     http.StatusOK,
			wantPermStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := middlewaremock.NewMockPermissionLoader(t)
			tt.setupLoader(loader)

			r := gin.New()
			r.Use(func(c *gin.Context) {
				tt.setupContext(c)
				c.Next()
			})
			r.Use(RequirePermission(loader, tt.permission))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantPermStored {
				assert.Contains(t, w.Body.String(), "ok")
			}
		})
	}
}

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/DenysonJ/financial-wallet/internal/mocks/useruci"
	useruc "github.com/DenysonJ/financial-wallet/internal/usecases/user"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newPasswordHandler(t *testing.T) (*PasswordHandler, *useruci.MockRepository) {
	t.Helper()
	mockRepo := useruci.NewMockRepository(t)
	setPasswordUC := useruc.NewSetPasswordUseCase(mockRepo)
	changePasswordUC := useruc.NewChangePasswordUseCase(mockRepo)
	h := NewPasswordHandler(setPasswordUC, changePasswordUC)
	return h, mockRepo
}

func setupPasswordRouterWithServiceKey(h *PasswordHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyServiceKey, "test-service")
		c.Next()
	}
	r.POST("/users/password", auth, h.SetPassword)
	return r
}

func setupPasswordRouterWithJWT(h *PasswordHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	}
	r.PUT("/users/password", auth, h.ChangePassword)
	return r
}

func setupPasswordRouterNoAuth(h *PasswordHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/users/password", h.ChangePassword)
	return r
}

// ---------------------------------------------------------------------------
// SetPassword
// ---------------------------------------------------------------------------

func TestPasswordHandler_SetPassword(t *testing.T) {
	userIDVal := uservo.NewID()
	userIDStr := userIDVal.String()

	email, _ := uservo.NewEmail("test@example.com")
	userWithNoPassword := &userdomain.User{
		ID:           userIDVal,
		Name:         "Test User",
		Email:        email,
		PasswordHash: "",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name       string
		body       any
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: map[string]string{
				"user_id":               userIDStr,
				"password":              "P@ssw0rd!",
				"password_confirmation": "P@ssw0rd!",
			},
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(userWithNoPassword, nil)
				m.On("UpdatePassword", mock.Anything, userIDVal, mock.AnythingOfType("string")).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid body",
			body:       "not json",
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "missing required fields",
			body: map[string]string{
				"user_id": userIDStr,
			},
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "user not found",
			body: map[string]string{
				"user_id":               userIDStr,
				"password":              "P@ssw0rd!",
				"password_confirmation": "P@ssw0rd!",
			},
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(nil, userdomain.ErrUserNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name: "password already set returns 409",
			body: map[string]string{
				"user_id":               userIDStr,
				"password":              "P@ssw0rd!",
				"password_confirmation": "P@ssw0rd!",
			},
			setupMock: func(m *useruci.MockRepository) {
				userWithPassword := *userWithNoPassword
				userWithPassword.PasswordHash = "existing-hash"
				m.On("FindByID", mock.Anything, userIDVal).Return(&userWithPassword, nil)
			},
			wantStatus: http.StatusConflict,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newPasswordHandler(t)
			tt.setupMock(mockRepo)

			r := setupPasswordRouterWithServiceKey(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/users/password", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// ChangePassword
// ---------------------------------------------------------------------------

func TestPasswordHandler_ChangePassword(t *testing.T) {
	userIDVal := uservo.NewID()
	userIDStr := userIDVal.String()

	email, _ := uservo.NewEmail("test@example.com")
	pw, _ := uservo.NewPassword("OldP@ss1!", 4) // low cost for tests
	passwordHash := pw.String()
	userWithPassword := &userdomain.User{
		ID:           userIDVal,
		Name:         "Test User",
		Email:        email,
		PasswordHash: passwordHash,
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name       string
		body       any
		authUserID string
		noAuth     bool
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: map[string]string{
				"current_password":          "OldP@ss1!",
				"new_password":              "NewP@ss2!",
				"new_password_confirmation": "NewP@ss2!",
			},
			authUserID: userIDStr,
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(userWithPassword, nil)
				m.On("UpdatePassword", mock.Anything, userIDVal, mock.AnythingOfType("string")).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid body",
			body:       "not json",
			authUserID: userIDStr,
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "no auth returns 401",
			body: map[string]string{
				"current_password":          "OldP@ss1!",
				"new_password":              "NewP@ss2!",
				"new_password_confirmation": "NewP@ss2!",
			},
			noAuth:     true,
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newPasswordHandler(t)
			tt.setupMock(mockRepo)

			var r *gin.Engine
			if tt.noAuth {
				r = setupPasswordRouterNoAuth(h)
			} else {
				r = setupPasswordRouterWithJWT(h, tt.authUserID)
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/users/password", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

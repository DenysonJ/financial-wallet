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
	"github.com/DenysonJ/financial-wallet/internal/mocks/authuci"
	authuc "github.com/DenysonJ/financial-wallet/internal/usecases/auth"
	authi "github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newAuthHandler(t *testing.T) (*AuthHandler, *authuci.MockUserRepository, *authuci.MockTokenService) {
	t.Helper()
	mockRepo := authuci.NewMockUserRepository(t)
	mockToken := authuci.NewMockTokenService(t)
	loginUC := authuc.NewLoginUseCase(mockRepo, mockToken)
	refreshUC := authuc.NewRefreshUseCase(mockToken)
	h := NewAuthHandler(loginUC, refreshUC)
	return h, mockRepo, mockToken
}

func setupAuthRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	return r
}

func newTestUser() *userdomain.User {
	email, _ := uservo.NewEmail("test@example.com")
	pw, _ := uservo.NewPassword("P@ssw0rd!", 4) // low cost for tests
	hash := pw.String()
	return &userdomain.User{
		ID:           uservo.NewID(),
		Name:         "Test User",
		Email:        email,
		PasswordHash: hash,
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupMock  func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, user *userdomain.User)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: map[string]string{"email": "test@example.com", "password": "P@ssw0rd!"},
			setupMock: func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, u *userdomain.User) {
				m.On("FindByEmail", mock.Anything, mock.AnythingOfType("vo.Email")).Return(u, nil)
				tok.On("GenerateAccessToken", u.ID.String()).Return("access-token", nil)
				tok.On("GenerateRefreshToken", u.ID.String()).Return("refresh-token", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			body:       "not json",
			setupMock:  func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, u *userdomain.User) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "missing email",
			body: map[string]string{"password": "P@ssw0rd!"},
			setupMock: func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, u *userdomain.User) {
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "user not found returns 401",
			body: map[string]string{"email": "unknown@example.com", "password": "P@ssw0rd!"},
			setupMock: func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, u *userdomain.User) {
				m.On("FindByEmail", mock.Anything, mock.AnythingOfType("vo.Email")).Return(nil, userdomain.ErrUserNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "wrong password returns 401",
			body: map[string]string{"email": "test@example.com", "password": "WrongPassword1!"},
			setupMock: func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, u *userdomain.User) {
				m.On("FindByEmail", mock.Anything, mock.AnythingOfType("vo.Email")).Return(u, nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "inactive user returns 401",
			body: map[string]string{"email": "test@example.com", "password": "P@ssw0rd!"},
			setupMock: func(m *authuci.MockUserRepository, tok *authuci.MockTokenService, u *userdomain.User) {
				inactiveUser := *u
				inactiveUser.Active = false
				m.On("FindByEmail", mock.Anything, mock.AnythingOfType("vo.Email")).Return(&inactiveUser, nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo, mockToken := newAuthHandler(t)
			u := newTestUser()
			tt.setupMock(mockRepo, mockToken, u)

			r := setupAuthRouter(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if !tt.wantErr {
				var resp map[string]any
				parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, parseErr)
				data, ok := resp["data"].(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, "access-token", data["access_token"])
				assert.Equal(t, "refresh-token", data["refresh_token"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestAuthHandler_Refresh(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupMock  func(tok *authuci.MockTokenService)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: map[string]string{"refresh_token": "valid-refresh-token"},
			setupMock: func(tok *authuci.MockTokenService) {
				tok.On("ValidateToken", "valid-refresh-token").Return(&authi.TokenClaims{
					UserID:    "user-123",
					TokenType: authi.TokenTypeRefresh,
				}, nil)
				tok.On("GenerateAccessToken", "user-123").Return("new-access-token", nil)
				tok.On("GenerateRefreshToken", "user-123").Return("new-refresh-token", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			body:       "not json",
			setupMock:  func(tok *authuci.MockTokenService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "missing refresh_token",
			body: map[string]string{},
			setupMock: func(tok *authuci.MockTokenService) {
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "invalid token returns 401",
			body: map[string]string{"refresh_token": "invalid-token"},
			setupMock: func(tok *authuci.MockTokenService) {
				tok.On("ValidateToken", "invalid-token").Return(nil, assert.AnError)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "access token type returns 401",
			body: map[string]string{"refresh_token": "access-token-value"},
			setupMock: func(tok *authuci.MockTokenService) {
				tok.On("ValidateToken", "access-token-value").Return(&authi.TokenClaims{
					UserID:    "user-123",
					TokenType: authi.TokenTypeAccess,
				}, nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, mockToken := newAuthHandler(t)
			tt.setupMock(mockToken)

			r := setupAuthRouter(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if !tt.wantErr {
				var resp map[string]any
				parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, parseErr)
				data, ok := resp["data"].(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, "new-access-token", data["access_token"])
				assert.Equal(t, "new-refresh-token", data["refresh_token"])
			}
		})
	}
}

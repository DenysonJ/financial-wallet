package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/DenysonJ/financial-wallet/internal/mocks/accountuci"
	accountuc "github.com/DenysonJ/financial-wallet/internal/usecases/account"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newAccountHandler(t *testing.T) (*AccountHandler, *accountuci.MockRepository) {
	t.Helper()
	mockRepo := accountuci.NewMockRepository(t)
	createUC := accountuc.NewCreateUseCase(mockRepo)
	getUC := accountuc.NewGetUseCase(mockRepo)
	listUC := accountuc.NewListUseCase(mockRepo)
	updateUC := accountuc.NewUpdateUseCase(mockRepo)
	deleteUC := accountuc.NewDeleteUseCase(mockRepo)
	h := NewAccountHandler(createUC, getUC, listUC, updateUC, deleteUC)
	return h, mockRepo
}

func setupAccountRouter(h *AccountHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/accounts", h.Create)
	r.GET("/accounts/:id", h.GetByID)
	r.GET("/accounts", h.List)
	r.PUT("/accounts/:id", h.Update)
	r.DELETE("/accounts/:id", h.Delete)
	return r
}

func setupAccountRouterWithAuth(h *AccountHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	}
	r.POST("/accounts", auth, h.Create)
	r.GET("/accounts/:id", auth, h.GetByID)
	r.GET("/accounts", auth, h.List)
	r.PUT("/accounts/:id", auth, h.Update)
	r.DELETE("/accounts/:id", auth, h.Delete)
	return r
}

func setupAccountRouterWithServiceKey(h *AccountHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyServiceKey, "test-service")
		c.Set(middleware.ContextKeyUserID, "admin-user-id")
		c.Next()
	}
	r.POST("/accounts", auth, h.Create)
	r.GET("/accounts/:id", auth, h.GetByID)
	r.GET("/accounts", auth, h.List)
	r.PUT("/accounts/:id", auth, h.Update)
	r.DELETE("/accounts/:id", auth, h.Delete)
	return r
}

func validAccountUUID() string {
	return pkgvo.NewID().String()
}

func newTestAccount(userID pkgvo.ID) *accountdomain.Account {
	return &accountdomain.Account{
		ID:          pkgvo.NewID(),
		UserID:      userID,
		Name:        "My Bank Account",
		Type:        accountvo.TypeBankAccount,
		Description: "Main checking account",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAccountHandler_Create(t *testing.T) {
	userID := validAccountUUID()

	tests := []struct {
		name       string
		body       any
		setupMock  func(m *accountuci.MockRepository)
		authUserID string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success",
			body:       map[string]string{"name": "Savings", "type": "bank_account", "description": "My savings"},
			authUserID: userID,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body",
			body:       "not json",
			authUserID: userID,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "missing required fields",
			body:       map[string]string{"description": "Only description"},
			authUserID: userID,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "no auth context returns 401",
			body:       map[string]string{"name": "Savings", "type": "bank_account"},
			authUserID: "",
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "invalid account type returns 400",
			body:       map[string]string{"name": "Savings", "type": "invalid_type"},
			authUserID: userID,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "repository error returns 500",
			body:       map[string]string{"name": "Savings", "type": "bank_account"},
			authUserID: userID,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return(assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			tt.setupMock(mockRepo)

			var r *gin.Engine
			if tt.authUserID != "" {
				r = setupAccountRouterWithAuth(h, tt.authUserID)
			} else {
				r = setupAccountRouter(h)
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantErr {
				var resp map[string]any
				parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, parseErr)
				assert.Contains(t, resp, "errors")
			} else {
				var resp map[string]any
				parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, parseErr)
				assert.Contains(t, resp, "data")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestAccountHandler_GetByID(t *testing.T) {
	userIDStr := validAccountUUID()
	userID, _ := pkgvo.ParseID(userIDStr)
	acct := newTestAccount(userID)
	accountIDStr := acct.ID.String()

	tests := []struct {
		name       string
		id         string
		setupMock  func(m *accountuci.MockRepository)
		authUserID string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success as owner",
			id:         accountIDStr,
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(acct, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			id:         accountIDStr,
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(nil, accountdomain.ErrAccountNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "invalid ID format",
			id:         "not-a-uuid",
			authUserID: userIDStr,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "service key request skips ownership",
			id:         accountIDStr,
			authUserID: "", // handled separately with service key router
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(acct, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			tt.setupMock(mockRepo)

			var r *gin.Engine
			if tt.authUserID != "" {
				r = setupAccountRouterWithAuth(h, tt.authUserID)
			} else {
				r = setupAccountRouterWithServiceKey(h)
			}

			req := httptest.NewRequest(http.MethodGet, "/accounts/"+tt.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantErr {
				var resp map[string]any
				parseErr := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, parseErr)
				assert.Contains(t, resp, "errors")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestAccountHandler_List(t *testing.T) {
	userIDStr := validAccountUUID()
	userID, _ := pkgvo.ParseID(userIDStr)
	acct := newTestAccount(userID)

	tests := []struct {
		name       string
		query      string
		setupMock  func(m *accountuci.MockRepository)
		authUserID string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success",
			query:      "?page=1&limit=10",
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).Return(&accountdomain.ListResult{
					Accounts: []*accountdomain.Account{acct},
					Total:    1,
					Page:     1,
					Limit:    10,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no auth returns 401",
			query:      "?page=1&limit=10",
			authUserID: "",
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "repository error returns 500",
			query:      "?page=1&limit=10",
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			tt.setupMock(mockRepo)

			var r *gin.Engine
			if tt.authUserID != "" {
				r = setupAccountRouterWithAuth(h, tt.authUserID)
			} else {
				r = setupAccountRouter(h)
			}

			req := httptest.NewRequest(http.MethodGet, "/accounts"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestAccountHandler_Update(t *testing.T) {
	userIDStr := validAccountUUID()
	userID, _ := pkgvo.ParseID(userIDStr)
	acct := newTestAccount(userID)
	accountIDStr := acct.ID.String()

	newName := "Updated Name"

	tests := []struct {
		name       string
		id         string
		body       any
		setupMock  func(m *accountuci.MockRepository)
		authUserID string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success",
			id:         accountIDStr,
			body:       map[string]any{"name": newName},
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(acct, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid body",
			id:         accountIDStr,
			body:       "not json",
			authUserID: userIDStr,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "not found",
			id:         accountIDStr,
			body:       map[string]any{"name": newName},
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(nil, accountdomain.ErrAccountNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "invalid ID format",
			id:         "not-a-uuid",
			body:       map[string]any{"name": newName},
			authUserID: userIDStr,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			tt.setupMock(mockRepo)

			r := setupAccountRouterWithAuth(h, tt.authUserID)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/accounts/"+tt.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAccountHandler_Delete(t *testing.T) {
	userIDStr := validAccountUUID()
	userID, _ := pkgvo.ParseID(userIDStr)
	acct := newTestAccount(userID)
	accountIDStr := acct.ID.String()

	tests := []struct {
		name       string
		id         string
		setupMock  func(m *accountuci.MockRepository)
		authUserID string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success",
			id:         accountIDStr,
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(acct, nil)
				m.On("Delete", mock.Anything, acct.ID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found",
			id:         accountIDStr,
			authUserID: userIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("FindByID", mock.Anything, acct.ID).Return(nil, accountdomain.ErrAccountNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "invalid ID format",
			id:         "not-a-uuid",
			authUserID: userIDStr,
			setupMock:  func(m *accountuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "service key request skips ownership check",
			id:   accountIDStr,
			setupMock: func(m *accountuci.MockRepository) {
				m.On("Delete", mock.Anything, acct.ID).Return(nil)
			},
			authUserID: "", // uses service key router
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newAccountHandler(t)
			tt.setupMock(mockRepo)

			var r *gin.Engine
			if tt.authUserID != "" {
				r = setupAccountRouterWithAuth(h, tt.authUserID)
			} else {
				r = setupAccountRouterWithServiceKey(h)
			}

			req := httptest.NewRequest(http.MethodDelete, "/accounts/"+tt.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

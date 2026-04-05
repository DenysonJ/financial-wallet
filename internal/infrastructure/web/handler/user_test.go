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

func newUserHandler(t *testing.T) (*UserHandler, *useruci.MockRepository) {
	t.Helper()
	mockRepo := useruci.NewMockRepository(t)
	createUC := useruc.NewCreateUseCase(mockRepo)
	getUC := useruc.NewGetUseCase(mockRepo)
	listUC := useruc.NewListUseCase(mockRepo)
	updateUC := useruc.NewUpdateUseCase(mockRepo)
	deleteUC := useruc.NewDeleteUseCase(mockRepo)
	h := NewUserHandler(createUC, getUC, listUC, updateUC, deleteUC, nil)
	return h, mockRepo
}

func setupUserRouterWithServiceKey(h *UserHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyServiceKey, "test-service")
		c.Next()
	}
	r.POST("/users", auth, h.Create)
	r.GET("/users/:id", auth, h.GetByID)
	r.GET("/users", auth, h.List)
	r.PUT("/users/:id", auth, h.Update)
	r.DELETE("/users/:id", auth, h.Delete)
	return r
}

func setupUserRouterWithJWT(h *UserHandler, userID string, roles []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		if roles != nil {
			c.Set("user_roles", roles)
		}
		c.Next()
	}
	r.POST("/users", auth, h.Create)
	r.GET("/users/:id", auth, h.GetByID)
	r.GET("/users", auth, h.List)
	r.PUT("/users/:id", auth, h.Update)
	r.DELETE("/users/:id", auth, h.Delete)
	return r
}

func newDomainUser(id uservo.ID) *userdomain.User {
	email, _ := uservo.NewEmail("user@example.com")
	return &userdomain.User{
		ID:        id,
		Name:      "Test User",
		Email:     email,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestUserHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: map[string]string{"name": "John Doe", "email": "john@example.com"},
			setupMock: func(m *useruci.MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body",
			body:       "not json",
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "missing required fields",
			body:       map[string]string{"name": "John"},
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "repository error",
			body: map[string]string{"name": "John Doe", "email": "john@example.com"},
			setupMock: func(m *useruci.MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newUserHandler(t)
			tt.setupMock(mockRepo)

			r := setupUserRouterWithServiceKey(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestUserHandler_GetByID(t *testing.T) {
	userIDVal := uservo.NewID()
	userIDStr := userIDVal.String()
	u := newDomainUser(userIDVal)

	tests := []struct {
		name       string
		id         string
		authUserID string
		roles      []string
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
	}{
		{
			name:       "success as owner",
			id:         userIDStr,
			authUserID: userIDStr,
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(u, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "forbidden for non-owner non-admin",
			id:         userIDStr,
			authUserID: validAccountUUID(),
			roles:      []string{"viewer"},
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can access other user",
			id:         userIDStr,
			authUserID: validAccountUUID(),
			roles:      []string{"admin"},
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(u, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			id:         userIDStr,
			authUserID: userIDStr,
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(nil, userdomain.ErrUserNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newUserHandler(t)
			tt.setupMock(mockRepo)

			r := setupUserRouterWithJWT(h, tt.authUserID, tt.roles)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestUserHandler_List(t *testing.T) {
	userIDVal := uservo.NewID()
	userIDStr := userIDVal.String()
	u := newDomainUser(userIDVal)

	tests := []struct {
		name       string
		query      string
		authUserID string
		roles      []string
		useService bool
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
	}{
		{
			name:       "admin sees full list",
			query:      "?page=1&limit=10",
			authUserID: validAccountUUID(),
			roles:      []string{"admin"},
			setupMock: func(m *useruci.MockRepository) {
				m.On("List", mock.Anything, mock.AnythingOfType("user.ListFilter")).Return(&userdomain.ListResult{
					Users: []*userdomain.User{u},
					Total: 1,
					Page:  1,
					Limit: 10,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "service key sees full list",
			query:      "?page=1&limit=10",
			useService: true,
			setupMock: func(m *useruci.MockRepository) {
				m.On("List", mock.Anything, mock.AnythingOfType("user.ListFilter")).Return(&userdomain.ListResult{
					Users: []*userdomain.User{u},
					Total: 1,
					Page:  1,
					Limit: 10,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-admin sees only self (calls GetUC)",
			query:      "",
			authUserID: userIDStr,
			roles:      []string{"viewer"},
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(u, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newUserHandler(t)
			tt.setupMock(mockRepo)

			var r *gin.Engine
			if tt.useService {
				r = setupUserRouterWithServiceKey(h)
			} else {
				r = setupUserRouterWithJWT(h, tt.authUserID, tt.roles)
			}

			req := httptest.NewRequest(http.MethodGet, "/users"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserHandler_Update(t *testing.T) {
	userIDVal := uservo.NewID()
	userIDStr := userIDVal.String()
	u := newDomainUser(userIDVal)
	newName := "Updated Name"

	tests := []struct {
		name       string
		id         string
		body       any
		authUserID string
		roles      []string
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
	}{
		{
			name:       "success as owner",
			id:         userIDStr,
			body:       map[string]any{"name": newName},
			authUserID: userIDStr,
			setupMock: func(m *useruci.MockRepository) {
				m.On("FindByID", mock.Anything, userIDVal).Return(u, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "forbidden for non-owner",
			id:         userIDStr,
			body:       map[string]any{"name": newName},
			authUserID: validAccountUUID(),
			roles:      []string{"viewer"},
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid body",
			id:         userIDStr,
			body:       "not json",
			authUserID: userIDStr,
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newUserHandler(t)
			tt.setupMock(mockRepo)

			r := setupUserRouterWithJWT(h, tt.authUserID, tt.roles)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.id, bytes.NewReader(bodyBytes))
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

func TestUserHandler_Delete(t *testing.T) {
	userIDVal := uservo.NewID()
	userIDStr := userIDVal.String()

	tests := []struct {
		name       string
		id         string
		authUserID string
		roles      []string
		setupMock  func(m *useruci.MockRepository)
		wantStatus int
	}{
		{
			name:       "success as owner",
			id:         userIDStr,
			authUserID: userIDStr,
			setupMock: func(m *useruci.MockRepository) {
				m.On("Delete", mock.Anything, userIDVal).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "forbidden for non-owner",
			id:         userIDStr,
			authUserID: validAccountUUID(),
			roles:      []string{"viewer"},
			setupMock:  func(m *useruci.MockRepository) {},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "not found",
			id:         userIDStr,
			authUserID: userIDStr,
			setupMock: func(m *useruci.MockRepository) {
				m.On("Delete", mock.Anything, userIDVal).Return(userdomain.ErrUserNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newUserHandler(t)
			tt.setupMock(mockRepo)

			r := setupUserRouterWithJWT(h, tt.authUserID, tt.roles)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/DenysonJ/financial-wallet/internal/mocks/roleuci"
	roleuc "github.com/DenysonJ/financial-wallet/internal/usecases/role"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newRoleHandler(t *testing.T) (*RoleHandler, *roleuci.MockRepository) {
	t.Helper()
	mockRepo := roleuci.NewMockRepository(t)
	createUC := roleuc.NewCreateUseCase(mockRepo)
	listUC := roleuc.NewListUseCase(mockRepo)
	deleteUC := roleuc.NewDeleteUseCase(mockRepo)
	assignUC := roleuc.NewAssignRoleUseCase(mockRepo)
	revokeUC := roleuc.NewRevokeRoleUseCase(mockRepo)
	h := NewRoleHandler(createUC, listUC, deleteUC, assignUC, revokeUC)
	return h, mockRepo
}

func setupRoleRouterWithServiceKey(h *RoleHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyServiceKey, "test-service")
		c.Next()
	}
	r.POST("/roles", auth, h.Create)
	r.GET("/roles", auth, h.List)
	r.DELETE("/roles/:id", auth, h.Delete)
	r.POST("/roles/:id/assign", auth, h.AssignRole)
	r.POST("/roles/:id/revoke", auth, h.RevokeRole)
	return r
}

func newDomainRole(id pkgvo.ID) *roledomain.Role {
	return &roledomain.Role{
		ID:          id,
		Name:        "admin",
		Description: "Admin role",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRoleHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setupMock  func(m *roleuci.MockRepository)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: map[string]string{"name": "editor", "description": "Editor role"},
			setupMock: func(m *roleuci.MockRepository) {
				m.On("FindByName", mock.Anything, "editor").Return(nil, roledomain.ErrRoleNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*role.Role")).Return(nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid body",
			body:       "not json",
			setupMock:  func(m *roleuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "missing name",
			body:       map[string]string{"description": "No name"},
			setupMock:  func(m *roleuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "duplicate name returns 409",
			body: map[string]string{"name": "admin", "description": "Admin role"},
			setupMock: func(m *roleuci.MockRepository) {
				role := newDomainRole(pkgvo.NewID())
				m.On("FindByName", mock.Anything, "admin").Return(role, nil)
			},
			wantStatus: http.StatusConflict,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newRoleHandler(t)
			tt.setupMock(mockRepo)

			r := setupRoleRouterWithServiceKey(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestRoleHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setupMock  func(m *roleuci.MockRepository)
		wantStatus int
	}{
		{
			name:  "success",
			query: "?page=1&limit=10",
			setupMock: func(m *roleuci.MockRepository) {
				m.On("List", mock.Anything, mock.AnythingOfType("role.ListFilter")).Return(&roledomain.ListResult{
					Roles: []*roledomain.Role{newDomainRole(pkgvo.NewID())},
					Total: 1,
					Page:  1,
					Limit: 10,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "repository error",
			query: "?page=1&limit=10",
			setupMock: func(m *roleuci.MockRepository) {
				m.On("List", mock.Anything, mock.AnythingOfType("role.ListFilter")).Return(nil, assert.AnError)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newRoleHandler(t)
			tt.setupMock(mockRepo)

			r := setupRoleRouterWithServiceKey(h)

			req := httptest.NewRequest(http.MethodGet, "/roles"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRoleHandler_Delete(t *testing.T) {
	roleID := pkgvo.NewID()
	roleIDStr := roleID.String()

	tests := []struct {
		name       string
		id         string
		setupMock  func(m *roleuci.MockRepository)
		wantStatus int
	}{
		{
			name: "success",
			id:   roleIDStr,
			setupMock: func(m *roleuci.MockRepository) {
				m.On("Delete", mock.Anything, roleID).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not found",
			id:   roleIDStr,
			setupMock: func(m *roleuci.MockRepository) {
				m.On("Delete", mock.Anything, roleID).Return(roledomain.ErrRoleNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID format",
			id:         "not-a-uuid",
			setupMock:  func(m *roleuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newRoleHandler(t)
			tt.setupMock(mockRepo)

			r := setupRoleRouterWithServiceKey(h)

			req := httptest.NewRequest(http.MethodDelete, "/roles/"+tt.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// AssignRole
// ---------------------------------------------------------------------------

func TestRoleHandler_AssignRole(t *testing.T) {
	roleID := pkgvo.NewID()
	roleIDStr := roleID.String()
	userID := pkgvo.NewID()
	userIDStr := userID.String()

	tests := []struct {
		name       string
		id         string
		body       any
		setupMock  func(m *roleuci.MockRepository)
		wantStatus int
	}{
		{
			name: "success",
			id:   roleIDStr,
			body: map[string]string{"user_id": userIDStr},
			setupMock: func(m *roleuci.MockRepository) {
				m.On("FindByID", mock.Anything, roleID).Return(newDomainRole(roleID), nil)
				m.On("AssignRole", mock.Anything, userID, roleID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid body",
			id:         roleIDStr,
			body:       "not json",
			setupMock:  func(m *roleuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "role already assigned returns 409",
			id:   roleIDStr,
			body: map[string]string{"user_id": userIDStr},
			setupMock: func(m *roleuci.MockRepository) {
				m.On("FindByID", mock.Anything, roleID).Return(newDomainRole(roleID), nil)
				m.On("AssignRole", mock.Anything, userID, roleID).Return(roledomain.ErrRoleAlreadyAssigned)
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newRoleHandler(t)
			tt.setupMock(mockRepo)

			r := setupRoleRouterWithServiceKey(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/roles/"+tt.id+"/assign", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// RevokeRole
// ---------------------------------------------------------------------------

func TestRoleHandler_RevokeRole(t *testing.T) {
	roleID := pkgvo.NewID()
	roleIDStr := roleID.String()
	userID := pkgvo.NewID()
	userIDStr := userID.String()

	tests := []struct {
		name       string
		id         string
		body       any
		setupMock  func(m *roleuci.MockRepository)
		wantStatus int
	}{
		{
			name: "success",
			id:   roleIDStr,
			body: map[string]string{"user_id": userIDStr},
			setupMock: func(m *roleuci.MockRepository) {
				m.On("FindByID", mock.Anything, roleID).Return(newDomainRole(roleID), nil)
				m.On("RevokeRole", mock.Anything, userID, roleID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid body",
			id:         roleIDStr,
			body:       "not json",
			setupMock:  func(m *roleuci.MockRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "role not assigned returns 404",
			id:   roleIDStr,
			body: map[string]string{"user_id": userIDStr},
			setupMock: func(m *roleuci.MockRepository) {
				m.On("FindByID", mock.Anything, roleID).Return(newDomainRole(roleID), nil)
				m.On("RevokeRole", mock.Anything, userID, roleID).Return(roledomain.ErrRoleNotAssigned)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo := newRoleHandler(t)
			tt.setupMock(mockRepo)

			r := setupRoleRouterWithServiceKey(h)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/roles/"+tt.id+"/revoke", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

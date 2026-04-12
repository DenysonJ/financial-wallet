package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/DenysonJ/financial-wallet/internal/mocks/stmtuci"
	stmtuc "github.com/DenysonJ/financial-wallet/internal/usecases/statement"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newStatementHandler(t *testing.T) (*StatementHandler, *stmtuci.MockRepository, *stmtuci.MockAccountRepository) {
	t.Helper()
	mockRepo := stmtuci.NewMockRepository(t)
	mockAccRepo := stmtuci.NewMockAccountRepository(t)
	createUC := stmtuc.NewCreateUseCase(mockRepo, mockAccRepo)
	reverseUC := stmtuc.NewReverseUseCase(mockRepo, mockAccRepo)
	getUC := stmtuc.NewGetUseCase(mockRepo, mockAccRepo)
	listUC := stmtuc.NewListUseCase(mockRepo, mockAccRepo)
	h := NewStatementHandler(createUC, reverseUC, getUC, listUC)
	return h, mockRepo, mockAccRepo
}

func setupStatementRouterWithAuth(h *StatementHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	}
	r.POST("/accounts/:id/statements", auth, h.Create)
	r.POST("/accounts/:id/statements/:statement_id/reverse", auth, h.Reverse)
	r.GET("/accounts/:id/statements", auth, h.List)
	r.GET("/accounts/:id/statements/:statement_id", auth, h.GetByID)
	return r
}

func makeActiveAccount(accountID, ownerID pkgvo.ID) *accountdomain.Account {
	now := time.Now()
	return &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}
}

func makeInactiveAccount(accountID, ownerID pkgvo.ID) *accountdomain.Account {
	now := time.Now()
	return &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: false, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}
}

func makeStatement(accountID pkgvo.ID, stmtType stmtvo.StatementType, amount int64) *stmtdomain.Statement {
	a, _ := stmtvo.NewAmount(amount)
	return &stmtdomain.Statement{
		ID: pkgvo.NewID(), AccountID: accountID, Type: stmtType,
		Amount: a, Description: "Test", BalanceAfter: 15000,
		CreatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestStatementHandler_Create(t *testing.T) {
	accountID := pkgvo.NewID()
	ownerID := pkgvo.NewID()

	tests := []struct {
		name       string
		body       any
		accountID  string
		ownerID    string
		setupMocks func(*stmtuci.MockRepository, *stmtuci.MockAccountRepository)
		wantStatus int
	}{
		{
			name:      "given valid credit when creating then returns 201",
			body:      map[string]any{"type": "credit", "amount": 5000, "description": "Salary"},
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), accountID).
					Return(int64(15000), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:      "given valid debit when creating then returns 201",
			body:      map[string]any{"type": "debit", "amount": 3000, "description": "Purchase"},
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), accountID).
					Return(int64(7000), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "given invalid JSON when creating then returns 400",
			body:       "not json",
			accountID:  accountID.String(),
			ownerID:    ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "given missing type when creating then returns 400",
			body:       map[string]any{"amount": 5000},
			accountID:  accountID.String(),
			ownerID:    ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "given nonexistent account when creating then returns 404",
			body:      map[string]any{"type": "credit", "amount": 5000},
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(nil, accountdomain.ErrAccountNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "given inactive account when creating then returns 422",
			body:      map[string]any{"type": "credit", "amount": 5000},
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeInactiveAccount(accountID, ownerID), nil)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "given invalid account ID when creating then returns 400",
			body:       map[string]any{"type": "credit", "amount": 5000},
			accountID:  "bad-id",
			ownerID:    ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "given repo failure when creating then returns 500",
			body:      map[string]any{"type": "credit", "amount": 5000},
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), accountID).
					Return(int64(0), fmt.Errorf("database error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo, mockAccRepo := newStatementHandler(t)
			tt.setupMocks(mockRepo, mockAccRepo)

			router := setupStatementRouterWithAuth(h, tt.ownerID)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/accounts/"+tt.accountID+"/statements", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Reverse
// ---------------------------------------------------------------------------

func TestStatementHandler_Reverse(t *testing.T) {
	accountID := pkgvo.NewID()
	ownerID := pkgvo.NewID()
	statementID := pkgvo.NewID()

	creditType, _ := stmtvo.NewStatementType("credit")
	amount, _ := stmtvo.NewAmount(5000)
	originalStmt := &stmtdomain.Statement{
		ID: statementID, AccountID: accountID, Type: creditType,
		Amount: amount, Description: "Original", BalanceAfter: 15000,
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		body        any
		accountID   string
		statementID string
		ownerID     string
		setupMocks  func(*stmtuci.MockRepository, *stmtuci.MockAccountRepository)
		wantStatus  int
	}{
		{
			name:        "given unreversed statement when reversing then returns 201",
			body:        map[string]any{"description": "Reversal"},
			accountID:   accountID.String(),
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("FindByID", mock.Anything, statementID).
					Return(originalStmt, nil)
				repo.On("HasReversal", mock.Anything, statementID).
					Return(false, nil)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), accountID).
					Return(int64(10000), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "given empty body when reversing then returns 201",
			body:        nil,
			accountID:   accountID.String(),
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("FindByID", mock.Anything, statementID).
					Return(originalStmt, nil)
				repo.On("HasReversal", mock.Anything, statementID).
					Return(false, nil)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), accountID).
					Return(int64(10000), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "given already reversed statement when reversing then returns 409",
			body:        nil,
			accountID:   accountID.String(),
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("FindByID", mock.Anything, statementID).
					Return(originalStmt, nil)
				repo.On("HasReversal", mock.Anything, statementID).
					Return(true, nil)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:        "given nonexistent statement when reversing then returns 404",
			body:        nil,
			accountID:   accountID.String(),
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("FindByID", mock.Anything, statementID).
					Return(nil, stmtdomain.ErrStatementNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "given invalid account ID when reversing then returns 400",
			body:        nil,
			accountID:   "bad-id",
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks:  func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo, mockAccRepo := newStatementHandler(t)
			tt.setupMocks(mockRepo, mockAccRepo)

			router := setupStatementRouterWithAuth(h, tt.ownerID)

			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			url := fmt.Sprintf("/accounts/%s/statements/%s/reverse", tt.accountID, tt.statementID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestStatementHandler_List(t *testing.T) {
	accountID := pkgvo.NewID()
	ownerID := pkgvo.NewID()

	tests := []struct {
		name       string
		accountID  string
		query      string
		ownerID    string
		setupMocks func(*stmtuci.MockRepository, *stmtuci.MockAccountRepository)
		wantStatus int
	}{
		{
			name:      "given valid account when listing then returns 200",
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("List", mock.Anything, mock.AnythingOfType("statement.ListFilter")).
					Return(&stmtdomain.ListResult{
						Statements: []*stmtdomain.Statement{},
						Total:      0, Page: 1, Limit: 20,
					}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "given type filter when listing then returns 200",
			accountID: accountID.String(),
			query:     "?type=credit&page=1&limit=10",
			ownerID:   ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("List", mock.Anything, mock.AnythingOfType("statement.ListFilter")).
					Return(&stmtdomain.ListResult{
						Statements: []*stmtdomain.Statement{},
						Total:      0, Page: 1, Limit: 10,
					}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "given nonexistent account when listing then returns 404",
			accountID: accountID.String(),
			ownerID:   ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(nil, accountdomain.ErrAccountNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "given invalid account ID when listing then returns 400",
			accountID:  "bad-id",
			ownerID:    ownerID.String(),
			setupMocks: func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo, mockAccRepo := newStatementHandler(t)
			tt.setupMocks(mockRepo, mockAccRepo)

			router := setupStatementRouterWithAuth(h, tt.ownerID)

			url := fmt.Sprintf("/accounts/%s/statements%s", tt.accountID, tt.query)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestStatementHandler_GetByID(t *testing.T) {
	accountID := pkgvo.NewID()
	ownerID := pkgvo.NewID()
	statementID := pkgvo.NewID()

	creditType, _ := stmtvo.NewStatementType("credit")

	tests := []struct {
		name        string
		accountID   string
		statementID string
		ownerID     string
		setupMocks  func(*stmtuci.MockRepository, *stmtuci.MockAccountRepository)
		wantStatus  int
	}{
		{
			name:        "given existing statement when getting then returns 200",
			accountID:   accountID.String(),
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("FindByID", mock.Anything, statementID).
					Return(makeStatement(accountID, creditType, 5000), nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "given nonexistent statement when getting then returns 404",
			accountID:   accountID.String(),
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks: func(repo *stmtuci.MockRepository, accRepo *stmtuci.MockAccountRepository) {
				accRepo.On("FindByID", mock.Anything, accountID).
					Return(makeActiveAccount(accountID, ownerID), nil)
				repo.On("FindByID", mock.Anything, statementID).
					Return(nil, stmtdomain.ErrStatementNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "given invalid statement ID when getting then returns 400",
			accountID:   accountID.String(),
			statementID: "bad-id",
			ownerID:     ownerID.String(),
			setupMocks:  func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "given invalid account ID when getting then returns 400",
			accountID:   "bad-id",
			statementID: statementID.String(),
			ownerID:     ownerID.String(),
			setupMocks:  func(_ *stmtuci.MockRepository, _ *stmtuci.MockAccountRepository) {},
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockRepo, mockAccRepo := newStatementHandler(t)
			tt.setupMocks(mockRepo, mockAccRepo)

			router := setupStatementRouterWithAuth(h, tt.ownerID)

			url := fmt.Sprintf("/accounts/%s/statements/%s", tt.accountID, tt.statementID)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

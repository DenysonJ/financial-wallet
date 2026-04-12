package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/db/postgres/repository"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	accountuc "github.com/DenysonJ/financial-wallet/internal/usecases/account"
	stmtuc "github.com/DenysonJ/financial-wallet/internal/usecases/statement"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// =============================================================================
// Setup helpers
// =============================================================================

func setupStatementTestRouter(userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	db := GetTestDB()
	accountRepo := repository.NewAccountRepository(db, db)
	stmtRepo := repository.NewStatementRepository(db, db)

	accountCreateUC := accountuc.NewCreateUseCase(accountRepo)
	accountGetUC := accountuc.NewGetUseCase(accountRepo)
	accountListUC := accountuc.NewListUseCase(accountRepo)
	accountUpdateUC := accountuc.NewUpdateUseCase(accountRepo)
	accountDeleteUC := accountuc.NewDeleteUseCase(accountRepo)
	accountHandler := handler.NewAccountHandler(accountCreateUC, accountGetUC, accountListUC, accountUpdateUC, accountDeleteUC)

	stmtCreateUC := stmtuc.NewCreateUseCase(stmtRepo, accountRepo)
	stmtReverseUC := stmtuc.NewReverseUseCase(stmtRepo, accountRepo)
	stmtGetUC := stmtuc.NewGetUseCase(stmtRepo, accountRepo)
	stmtListUC := stmtuc.NewListUseCase(stmtRepo, accountRepo)
	stmtHandler := handler.NewStatementHandler(stmtCreateUC, stmtReverseUC, stmtGetUC, stmtListUC)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	})

	r.POST("/accounts", accountHandler.Create)
	r.GET("/accounts/:id", accountHandler.GetByID)
	r.POST("/accounts/:id/statements", stmtHandler.Create)
	r.GET("/accounts/:id/statements", stmtHandler.List)
	r.GET("/accounts/:id/statements/:statement_id", stmtHandler.GetByID)
	r.POST("/accounts/:id/statements/:statement_id/reverse", stmtHandler.Reverse)

	return r
}

func cleanupStatements() error {
	// Delete in FK dependency order: statements → accounts (users are shared, don't delete)
	if _, execErr := testDB.Exec("DELETE FROM statements"); execErr != nil {
		return execErr
	}
	_, execErr := testDB.Exec("DELETE FROM accounts")
	return execErr
}

func seedTestUser(t *testing.T, userID string) {
	t.Helper()
	_, insertErr := testDB.Exec(
		`INSERT INTO users (id, name, email, active, created_at, updated_at)
		 VALUES ($1, 'E2E Test User', $2, true, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		userID, userID+"@test.local",
	)
	require.NoError(t, insertErr)
}

func createTestAccount(t *testing.T, router *gin.Engine) string {
	t.Helper()
	body := `{"name": "Test Account", "type": "bank_account", "description": "E2E test"}`
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create account: %s", w.Body.String())
	return extractData(t, w.Body.Bytes())["id"].(string)
}

func createStatement(t *testing.T, router *gin.Engine, accountID, stmtType string, amount int64, description string) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"type": stmtType, "amount": amount, "description": description,
	})
	req := httptest.NewRequest(http.MethodPost, "/accounts/"+accountID+"/statements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create statement: %s", w.Body.String())
	return extractData(t, w.Body.Bytes())
}

// setupE2E is a convenience that cleans up, seeds a user, builds a router, and creates an account.
func setupE2E(t *testing.T) (*gin.Engine, string, string) {
	t.Helper()
	require.NoError(t, cleanupStatements())
	userID := vo.NewID().String()
	seedTestUser(t, userID)
	router := setupStatementTestRouter(userID)
	accountID := createTestAccount(t, router)
	return router, userID, accountID
}

// =============================================================================
// Create Statement — table-driven
// =============================================================================

func TestE2E_CreateStatement(t *testing.T) {
	tests := []struct {
		name             string
		seedCredits      []int64 // credits created before the test statement
		stmtType         string
		amount           int64
		description      string
		wantBalanceAfter int64
		wantDBBalance    int64
	}{
		{
			name:             "credit increases balance",
			stmtType:         "credit",
			amount:           10000,
			description:      "Salary deposit",
			wantBalanceAfter: 10000,
			wantDBBalance:    10000,
		},
		{
			name:             "debit decreases balance",
			seedCredits:      []int64{10000},
			stmtType:         "debit",
			amount:           3000,
			description:      "Purchase",
			wantBalanceAfter: 7000,
			wantDBBalance:    7000,
		},
		{
			name:             "debit allows negative balance (overdraft)",
			stmtType:         "debit",
			amount:           5000,
			description:      "Overdraft",
			wantBalanceAfter: -5000,
			wantDBBalance:    -5000,
		},
		{
			name:             "credit with empty description",
			stmtType:         "credit",
			amount:           1,
			description:      "",
			wantBalanceAfter: 1,
			wantDBBalance:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, accountID := setupE2E(t)

			for _, seedAmount := range tt.seedCredits {
				createStatement(t, router, accountID, "credit", seedAmount, "seed")
			}

			data := createStatement(t, router, accountID, tt.stmtType, tt.amount, tt.description)

			assert.NotEmpty(t, data["id"])
			assert.Equal(t, accountID, data["account_id"])
			assert.Equal(t, tt.stmtType, data["type"])
			assert.Equal(t, float64(tt.amount), data["amount"])
			assert.Equal(t, tt.description, data["description"])
			assert.Equal(t, float64(tt.wantBalanceAfter), data["balance_after"])
			assert.NotEmpty(t, data["created_at"])
			assert.Nil(t, data["reference_id"])

			var dbBalance int64
			dbErr := GetTestDB().Get(&dbBalance, "SELECT balance FROM accounts WHERE id = $1", accountID)
			require.NoError(t, dbErr)
			assert.Equal(t, tt.wantDBBalance, dbBalance)
		})
	}
}

// =============================================================================
// Create Statement — error cases (table-driven)
// =============================================================================

func TestE2E_CreateStatement_Errors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid type",
			body:       `{"type": "transfer", "amount": 1000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero amount",
			body:       `{"type": "credit", "amount": 0}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative amount",
			body:       `{"type": "credit", "amount": -100}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing type",
			body:       `{"amount": 1000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, accountID := setupE2E(t)

			req := httptest.NewRequest(http.MethodPost, "/accounts/"+accountID+"/statements", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// =============================================================================
// Reversal
// =============================================================================

func TestE2E_ReverseStatement(t *testing.T) {
	router, _, accountID := setupE2E(t)

	original := createStatement(t, router, accountID, "credit", 5000, "Payment received")
	originalID := original["id"].(string)

	body := `{"description": "Payment reversal"}`
	req := httptest.NewRequest(http.MethodPost, "/accounts/"+accountID+"/statements/"+originalID+"/reverse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "reverse: %s", w.Body.String())

	reversal := extractData(t, w.Body.Bytes())

	assert.NotEmpty(t, reversal["id"])
	assert.Equal(t, accountID, reversal["account_id"])
	assert.Equal(t, "debit", reversal["type"])
	assert.Equal(t, float64(5000), reversal["amount"])
	assert.Equal(t, originalID, reversal["reference_id"])
	assert.Equal(t, float64(0), reversal["balance_after"])

	var balance int64
	dbErr := GetTestDB().Get(&balance, "SELECT balance FROM accounts WHERE id = $1", accountID)
	require.NoError(t, dbErr)
	assert.Equal(t, int64(0), balance)
}

func TestE2E_DoubleReversalReturns409(t *testing.T) {
	router, _, accountID := setupE2E(t)

	original := createStatement(t, router, accountID, "credit", 5000, "Payment")
	originalID := original["id"].(string)

	// First reversal — 201
	req := httptest.NewRequest(http.MethodPost, "/accounts/"+accountID+"/statements/"+originalID+"/reverse", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Second reversal — 409
	req = httptest.NewRequest(http.MethodPost, "/accounts/"+accountID+"/statements/"+originalID+"/reverse", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// =============================================================================
// Get Statement — table-driven
// =============================================================================

func TestE2E_GetStatement(t *testing.T) {
	router, _, accountID := setupE2E(t)

	created := createStatement(t, router, accountID, "credit", 7500, "Transfer in")
	stmtID := created["id"].(string)

	tests := []struct {
		name        string
		accountID   string
		statementID string
		wantStatus  int
		wantType    string
	}{
		{
			name:        "success",
			accountID:   accountID,
			statementID: stmtID,
			wantStatus:  http.StatusOK,
			wantType:    "credit",
		},
		{
			name:        "not found — fake statement ID",
			accountID:   accountID,
			statementID: vo.NewID().String(),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "cross-account denied — fake account ID",
			accountID:   vo.NewID().String(),
			statementID: stmtID,
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/accounts/"+tt.accountID+"/statements/"+tt.statementID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantStatus == http.StatusOK {
				data := extractData(t, w.Body.Bytes())
				assert.Equal(t, stmtID, data["id"])
				assert.Equal(t, tt.wantType, data["type"])
				assert.Equal(t, float64(7500), data["amount"])
			}
		})
	}
}

// =============================================================================
// List Statements — table-driven
// =============================================================================

func TestE2E_ListStatements(t *testing.T) {
	router, _, accountID := setupE2E(t)

	createStatement(t, router, accountID, "credit", 10000, "Deposit 1")
	createStatement(t, router, accountID, "credit", 5000, "Deposit 2")
	createStatement(t, router, accountID, "debit", 3000, "Purchase")

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantTotal float64
	}{
		{
			name:      "all statements",
			query:     "",
			wantCount: 3,
			wantTotal: 3,
		},
		{
			name:      "filter by credit",
			query:     "?type=credit",
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:      "filter by debit",
			query:     "?type=debit",
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:      "pagination page=1 limit=2",
			query:     "?page=1&limit=2",
			wantCount: 2,
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID+"/statements"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

			data := response["data"].(map[string]interface{})
			items := data["data"].([]interface{})
			pagination := data["pagination"].(map[string]interface{})

			assert.Len(t, items, tt.wantCount)
			assert.Equal(t, tt.wantTotal, pagination["total"])
		})
	}

	// Verify reverse chronological order (newest first = debit)
	t.Run("reverse chronological order", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID+"/statements", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		items := response["data"].(map[string]interface{})["data"].([]interface{})
		first := items[0].(map[string]interface{})
		assert.Equal(t, "debit", first["type"])
	})
}

// =============================================================================
// Account balance in GET response (REQ-6)
// =============================================================================

func TestE2E_AccountBalanceInGetResponse(t *testing.T) {
	router, _, accountID := setupE2E(t)

	createStatement(t, router, accountID, "credit", 15000, "Big deposit")

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := extractData(t, w.Body.Bytes())
	assert.Equal(t, float64(15000), data["balance"])
}

// =============================================================================
// Transactional safety
// =============================================================================

func TestE2E_AtomicBalanceUpdate(t *testing.T) {
	router, _, accountID := setupE2E(t)

	ops := []struct {
		stmtType string
		amount   int64
	}{
		{"credit", 10000},
		{"credit", 5000},
		{"debit", 3000},
		{"debit", 7000},
		{"credit", 2000},
	}
	for _, op := range ops {
		createStatement(t, router, accountID, op.stmtType, op.amount, "op")
	}

	// 10000 + 5000 - 3000 - 7000 + 2000 = 7000
	var balance int64
	require.NoError(t, GetTestDB().Get(&balance, "SELECT balance FROM accounts WHERE id = $1", accountID))
	assert.Equal(t, int64(7000), balance)

	var count int
	require.NoError(t, GetTestDB().Get(&count, "SELECT COUNT(*) FROM statements WHERE account_id = $1", accountID))
	assert.Equal(t, 5, count)
}

func TestE2E_BalanceAfterIsConsistent(t *testing.T) {
	router, _, accountID := setupE2E(t)

	steps := []struct {
		stmtType         string
		amount           int64
		wantBalanceAfter float64
	}{
		{"credit", 10000, 10000},
		{"debit", 3000, 7000},
		{"credit", 500, 7500},
	}
	for _, step := range steps {
		data := createStatement(t, router, accountID, step.stmtType, step.amount, "step")
		assert.Equal(t, step.wantBalanceAfter, data["balance_after"], "after %s %d", step.stmtType, step.amount)
	}
}

// =============================================================================
// Ownership enforcement
// =============================================================================

func TestE2E_OwnershipEnforcement(t *testing.T) {
	require.NoError(t, cleanupStatements())

	ownerID := vo.NewID().String()
	otherUserID := vo.NewID().String()
	seedTestUser(t, ownerID)
	seedTestUser(t, otherUserID)

	ownerRouter := setupStatementTestRouter(ownerID)
	accountID := createTestAccount(t, ownerRouter)
	created := createStatement(t, ownerRouter, accountID, "credit", 5000, "Owner deposit")
	stmtID := created["id"].(string)

	otherRouter := setupStatementTestRouter(otherUserID)

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID+"/statements/"+stmtID, nil)
	w := httptest.NewRecorder()
	otherRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "non-owner should get 404")
}

// =============================================================================
// Immutability — no PUT/PATCH/DELETE routes
// =============================================================================

func TestE2E_StatementsAreImmutable(t *testing.T) {
	router, _, accountID := setupE2E(t)

	created := createStatement(t, router, accountID, "credit", 10000, "Deposit")
	stmtID := created["id"].(string)

	tests := []struct {
		name   string
		method string
	}{
		{"PUT not allowed", http.MethodPut},
		{"PATCH not allowed", http.MethodPatch},
		{"DELETE not allowed", http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/accounts/"+accountID+"/statements/"+stmtID, bytes.NewBufferString(`{}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.NotEqual(t, http.StatusOK, w.Code)
			assert.NotEqual(t, http.StatusNoContent, w.Code)
		})
	}
}

// =============================================================================
// Pagination metadata
// =============================================================================

func TestE2E_ListStatements_PaginationMetadata(t *testing.T) {
	router, _, accountID := setupE2E(t)

	for i := 1; i <= 5; i++ {
		createStatement(t, router, accountID, "credit", int64(i*1000), fmt.Sprintf("Deposit %d", i))
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID+"/statements?page=1&limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	data := response["data"].(map[string]interface{})
	items := data["data"].([]interface{})
	pagination := data["pagination"].(map[string]interface{})

	assert.Len(t, items, 2)
	assert.Equal(t, float64(5), pagination["total"])
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(2), pagination["limit"])
	assert.Equal(t, float64(3), pagination["total_pages"])
}

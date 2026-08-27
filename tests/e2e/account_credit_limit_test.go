package e2e

import (
	"bytes"
	"database/sql"
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
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// setupAccountRouter monta apenas a fatia de accounts contra o Postgres real —
// é o que exercita a coluna credit_limit e a CHECK da migration de ponta a ponta.
func setupAccountRouter(userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db := GetTestDB()

	accountRepo := repository.NewAccountRepository(db, db)
	accountHandler := handler.NewAccountHandler(
		accountuc.NewCreateUseCase(accountRepo),
		accountuc.NewGetUseCase(accountRepo),
		accountuc.NewListUseCase(accountRepo),
		accountuc.NewUpdateUseCase(accountRepo),
		accountuc.NewDeleteUseCase(accountRepo),
	)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	})

	r.POST("/accounts", accountHandler.Create)
	r.GET("/accounts", accountHandler.List)
	r.GET("/accounts/:id", accountHandler.GetByID)
	r.PUT("/accounts/:id", accountHandler.Update)

	return r
}

func putJSONReq(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestE2E_AccountCreditLimit cobre o ciclo completo do limite de crédito contra
// o banco real: criação, leitura, atualização e as recusas por tipo e por valor.
func TestE2E_AccountCreditLimit(t *testing.T) {
	userID := vo.NewID().String()
	seedTestUser(t, userID)
	t.Cleanup(func() { cleanupUserData(t, userID) })
	router := setupAccountRouter(userID)

	var creditCardID string

	t.Run("GIVEN a credit card WHEN created with a valid credit limit THEN 201 and the column is persisted (UC-01)", func(t *testing.T) {
		w := postJSON(t, router, "/accounts", `{"name":"Cartao Nubank","type":"credit_card","credit_limit":500000}`)
		require.Equal(t, http.StatusCreated, w.Code, "create credit card: %s", w.Body.String())
		creditCardID = extractData(t, w.Body.Bytes())["id"].(string)

		var persisted sql.NullInt64
		queryErr := testDB.Get(&persisted, "SELECT credit_limit FROM accounts WHERE id = $1", creditCardID)
		require.NoError(t, queryErr)
		assert.True(t, persisted.Valid)
		assert.Equal(t, int64(500000), persisted.Int64)
	})

	t.Run("GIVEN a credit card WHEN read THEN credit_limit comes back in cents (UC-01, RN-15)", func(t *testing.T) {
		w := getJSON(t, router, "/accounts/"+creditCardID)
		require.Equal(t, http.StatusOK, w.Code, "get account: %s", w.Body.String())
		assert.Equal(t, float64(500000), extractData(t, w.Body.Bytes())["credit_limit"])
	})

	t.Run("GIVEN a credit card WHEN created without a credit limit THEN 422 (UC-02)", func(t *testing.T) {
		w := postJSON(t, router, "/accounts", `{"name":"Cartao sem limite","type":"credit_card"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, "body: %s", w.Body.String())
	})

	t.Run("GIVEN a bank account WHEN created with a credit limit THEN 422 (UC-03)", func(t *testing.T) {
		w := postJSON(t, router, "/accounts", `{"name":"Conta","type":"bank_account","credit_limit":100000}`)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, "body: %s", w.Body.String())
	})

	t.Run("GIVEN a credit card WHEN created with an invalid credit limit THEN 400 (UC-04)", func(t *testing.T) {
		w := postJSON(t, router, "/accounts", `{"name":"Cartao","type":"credit_card","credit_limit":0}`)
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
	})

	t.Run("GIVEN a bank account WHEN created without a credit limit THEN the column stays NULL (UC-10)", func(t *testing.T) {
		bankID := createTestAccount(t, router)

		var persisted sql.NullInt64
		queryErr := testDB.Get(&persisted, "SELECT credit_limit FROM accounts WHERE id = $1", bankID)
		require.NoError(t, queryErr)
		assert.False(t, persisted.Valid, "conta sem limite deve gravar NULL, não zero")

		w := getJSON(t, router, "/accounts/"+bankID)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Nil(t, extractData(t, w.Body.Bytes())["credit_limit"])

		t.Run("AND WHEN a credit limit is set on it THEN 422 (UC-06)", func(t *testing.T) {
			limitW := putJSONReq(t, router, "/accounts/"+bankID, `{"credit_limit":50000}`)
			assert.Equal(t, http.StatusUnprocessableEntity, limitW.Code, "body: %s", limitW.Body.String())
		})
	})

	t.Run("GIVEN a credit card WHEN only the credit limit is updated THEN 200 and it is replaced (UC-05)", func(t *testing.T) {
		w := putJSONReq(t, router, "/accounts/"+creditCardID, `{"credit_limit":800000}`)
		require.Equal(t, http.StatusOK, w.Code, "update limit: %s", w.Body.String())
		assert.Equal(t, float64(800000), extractData(t, w.Body.Bytes())["credit_limit"])

		var persisted sql.NullInt64
		require.NoError(t, testDB.Get(&persisted, "SELECT credit_limit FROM accounts WHERE id = $1", creditCardID))
		assert.Equal(t, int64(800000), persisted.Int64)
	})

	t.Run("GIVEN a credit card WHEN the update omits the credit limit THEN the limit is preserved (UC-08)", func(t *testing.T) {
		w := putJSONReq(t, router, "/accounts/"+creditCardID, `{"name":"Cartao Roxinho"}`)
		require.Equal(t, http.StatusOK, w.Code, "update name: %s", w.Body.String())

		data := extractData(t, w.Body.Bytes())
		assert.Equal(t, "Cartao Roxinho", data["name"])
		assert.Equal(t, float64(800000), data["credit_limit"])
	})

	t.Run("GIVEN a credit card WHEN the update sends credit_limit null THEN the limit is preserved (UC-08, RN-09)", func(t *testing.T) {
		w := putJSONReq(t, router, "/accounts/"+creditCardID, `{"credit_limit":null}`)
		require.Equal(t, http.StatusOK, w.Code, "update null: %s", w.Body.String())
		assert.Equal(t, float64(800000), extractData(t, w.Body.Bytes())["credit_limit"])
	})

	t.Run("GIVEN a credit card WHEN the limit is reduced below the amount used THEN 200 (UC-07, RN-11)", func(t *testing.T) {
		// Fatura em aberto: saldo devedor de R$ 3.000 no cartão.
		_, updateErr := testDB.Exec("UPDATE accounts SET balance = -300000 WHERE id = $1", creditCardID)
		require.NoError(t, updateErr)

		w := putJSONReq(t, router, "/accounts/"+creditCardID, `{"credit_limit":100000}`)
		require.Equal(t, http.StatusOK, w.Code, "reduce limit: %s", w.Body.String())

		data := extractData(t, w.Body.Bytes())
		assert.Equal(t, float64(100000), data["credit_limit"])

		// O saldo devedor não é tocado pela alteração de limite (INV-04, RN-13).
		var balance int64
		require.NoError(t, testDB.Get(&balance, "SELECT balance FROM accounts WHERE id = $1", creditCardID))
		assert.Equal(t, int64(-300000), balance)
	})

	t.Run("GIVEN a credit card WHEN a limit above the ceiling is sent THEN 400 (UC-04)", func(t *testing.T) {
		w := putJSONReq(t, router, "/accounts/"+creditCardID, `{"credit_limit":1000000000001}`)
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
	})

	t.Run("GIVEN an account of another user WHEN its limit is updated THEN 404 (UC-09)", func(t *testing.T) {
		otherUserID := vo.NewID().String()
		seedTestUser(t, otherUserID)
		t.Cleanup(func() { cleanupUserData(t, otherUserID) })
		otherCardID := extractData(t, postJSON(t, setupAccountRouter(otherUserID), "/accounts",
			`{"name":"Cartao do outro","type":"credit_card","credit_limit":700000}`).Body.Bytes())["id"].(string)

		w := putJSONReq(t, router, "/accounts/"+otherCardID, `{"credit_limit":900000}`)
		assert.Equal(t, http.StatusNotFound, w.Code, "body: %s", w.Body.String())
	})
}

// TestE2E_AccountCreditLimit_DatabaseConstraint prova que a CHECK da migration
// é rede de segurança real: mesmo por fora da aplicação, o banco recusa estado
// que viole a invariante tipo × limite.
func TestE2E_AccountCreditLimit_DatabaseConstraint(t *testing.T) {
	userID := vo.NewID().String()
	seedTestUser(t, userID)
	t.Cleanup(func() { cleanupUserData(t, userID) })
	router := setupAccountRouter(userID)

	creditCardID := extractData(t, postJSON(t, router, "/accounts",
		`{"name":"Cartao","type":"credit_card","credit_limit":500000}`).Body.Bytes())["id"].(string)
	bankID := createTestAccount(t, router)

	t.Run("GIVEN a credit card WHEN a direct UPDATE clears the limit THEN the database rejects it (INV-01)", func(t *testing.T) {
		_, execErr := testDB.Exec("UPDATE accounts SET credit_limit = NULL WHERE id = $1", creditCardID)
		assert.Error(t, execErr)
		assert.Contains(t, execErr.Error(), "accounts_credit_limit_type_check")
	})

	t.Run("GIVEN a bank account WHEN a direct UPDATE sets a limit THEN the database rejects it (INV-02)", func(t *testing.T) {
		_, execErr := testDB.Exec("UPDATE accounts SET credit_limit = 100000 WHERE id = $1", bankID)
		assert.Error(t, execErr)
		assert.Contains(t, execErr.Error(), "accounts_credit_limit_type_check")
	})

	t.Run("GIVEN a credit card WHEN a direct UPDATE exceeds the ceiling THEN the database rejects it (INV-03)", func(t *testing.T) {
		_, execErr := testDB.Exec("UPDATE accounts SET credit_limit = 1000000000001 WHERE id = $1", creditCardID)
		assert.Error(t, execErr)
		assert.Contains(t, execErr.Error(), "accounts_credit_limit_type_check")
	})
}

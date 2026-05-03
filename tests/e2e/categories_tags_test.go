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
	categoryuc "github.com/DenysonJ/financial-wallet/internal/usecases/category"
	stmtuc "github.com/DenysonJ/financial-wallet/internal/usecases/statement"
	taguc "github.com/DenysonJ/financial-wallet/internal/usecases/tag"
	"github.com/DenysonJ/financial-wallet/pkg/ofx"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// setupCategoriesTagsRouter wires accounts + categories + tags + statements
// (with category/tag dependencies injected) for the REQ-13 e2e scenario.
func setupCategoriesTagsRouter(userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db := GetTestDB()

	accountRepo := repository.NewAccountRepository(db, db)
	stmtRepo := repository.NewStatementRepository(db, db)
	categoryRepo := repository.NewCategoryRepository(db, db)
	tagRepo := repository.NewTagRepository(db, db)

	accountCreateUC := accountuc.NewCreateUseCase(accountRepo)
	accountGetUC := accountuc.NewGetUseCase(accountRepo)
	accountListUC := accountuc.NewListUseCase(accountRepo)
	accountUpdateUC := accountuc.NewUpdateUseCase(accountRepo)
	accountDeleteUC := accountuc.NewDeleteUseCase(accountRepo)
	accountHandler := handler.NewAccountHandler(accountCreateUC, accountGetUC, accountListUC, accountUpdateUC, accountDeleteUC)

	categoryCreateUC := categoryuc.NewCreateUseCase(categoryRepo)
	categoryListUC := categoryuc.NewListUseCase(categoryRepo)
	categoryUpdateUC := categoryuc.NewUpdateUseCase(categoryRepo)
	categoryDeleteUC := categoryuc.NewDeleteUseCase(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categoryCreateUC, categoryListUC, categoryUpdateUC, categoryDeleteUC)

	tagCreateUC := taguc.NewCreateUseCase(tagRepo)
	tagListUC := taguc.NewListUseCase(tagRepo)
	tagUpdateUC := taguc.NewUpdateUseCase(tagRepo)
	tagDeleteUC := taguc.NewDeleteUseCase(tagRepo)
	tagHandler := handler.NewTagHandler(tagCreateUC, tagListUC, tagUpdateUC, tagDeleteUC)

	stmtCreateUC := stmtuc.NewCreateUseCase(stmtRepo, accountRepo).
		WithCategoryRepo(categoryRepo).WithTagRepo(tagRepo)
	stmtReverseUC := stmtuc.NewReverseUseCase(stmtRepo, accountRepo)
	stmtGetUC := stmtuc.NewGetUseCase(stmtRepo, accountRepo)
	stmtListUC := stmtuc.NewListUseCase(stmtRepo, accountRepo)
	stmtImportUC := stmtuc.NewImportUseCase(stmtRepo, accountRepo, ofx.NewParser())
	stmtHandler := handler.NewStatementHandler(stmtCreateUC, stmtReverseUC, stmtGetUC, stmtListUC, stmtImportUC)

	stmtUpdateCategoryUC := stmtuc.NewUpdateCategoryUseCase(stmtRepo, accountRepo, categoryRepo)
	stmtReplaceTagsUC := stmtuc.NewReplaceTagsUseCase(stmtRepo, accountRepo, tagRepo)
	stmtMetaHandler := handler.NewStatementMetadataHandler(stmtUpdateCategoryUC, stmtReplaceTagsUC)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	})

	r.POST("/accounts", accountHandler.Create)
	r.POST("/accounts/:id/statements", stmtHandler.Create)
	r.GET("/accounts/:id/statements", stmtHandler.List)
	r.POST("/accounts/:id/statements/:statement_id/reverse", stmtHandler.Reverse)

	r.POST("/categories", categoryHandler.Create)
	r.GET("/categories", categoryHandler.List)
	r.PATCH("/categories/:id", categoryHandler.Update)
	r.DELETE("/categories/:id", categoryHandler.Delete)

	r.POST("/tags", tagHandler.Create)
	r.GET("/tags", tagHandler.List)

	r.PATCH("/statements/:id/category", stmtMetaHandler.UpdateCategory)
	r.PUT("/statements/:id/tags", stmtMetaHandler.ReplaceTags)

	return r
}

func cleanupCategoriesTags(t *testing.T) {
	t.Helper()
	// Order respects FK constraints: statement_tags → statements → accounts;
	// user-scoped categories/tags after statements (RESTRICT FK).
	for _, q := range []string{
		"DELETE FROM statement_tags",
		"DELETE FROM statements",
		"DELETE FROM accounts",
		"DELETE FROM categories WHERE user_id IS NOT NULL",
		"DELETE FROM tags WHERE user_id IS NOT NULL",
	} {
		_, execErr := testDB.Exec(q)
		require.NoError(t, execErr, "cleanup: %s", q)
	}
}

// =============================================================================
// REQ-13 end-to-end scenario
// =============================================================================

func TestE2E_CategoriesAndTags_FullFlow(t *testing.T) {
	cleanupCategoriesTags(t)

	userID := vo.NewID().String()
	seedTestUser(t, userID)
	router := setupCategoriesTagsRouter(userID)

	accountID := createTestAccount(t, router)

	// 1. Create a custom debit category.
	catBody := `{"name": "Mercado", "type": "debit"}`
	catW := postJSON(t, router, "/categories", catBody)
	require.Equal(t, http.StatusCreated, catW.Code, "create category: %s", catW.Body.String())
	categoryID := extractData(t, catW.Body.Bytes())["id"].(string)

	// 2. Create two custom tags.
	tag1ID := createTag(t, router, "viagem-2026")
	tag2ID := createTag(t, router, "recurring")

	// 3. Create a debit statement with the category + tags.
	stmtBody, _ := json.Marshal(map[string]interface{}{
		"type":        "debit",
		"amount":      2500,
		"description": "Compras do mês",
		"category_id": categoryID,
		"tag_ids":     []string{tag1ID, tag2ID},
	})
	stmtReq := httptest.NewRequest(http.MethodPost, "/accounts/"+accountID+"/statements", bytes.NewReader(stmtBody))
	stmtReq.Header.Set("Content-Type", "application/json")
	stmtW := httptest.NewRecorder()
	router.ServeHTTP(stmtW, stmtReq)
	require.Equal(t, http.StatusCreated, stmtW.Code, "create statement: %s", stmtW.Body.String())

	stmtData := extractData(t, stmtW.Body.Bytes())
	stmtID := stmtData["id"].(string)
	cat := stmtData["category"].(map[string]interface{})
	assert.Equal(t, categoryID, cat["id"])
	assert.Equal(t, "Mercado", cat["name"])
	assert.Equal(t, "debit", cat["type"])
	tags, _ := stmtData["tags"].([]interface{})
	assert.Len(t, tags, 2, "statement must carry 2 tags")

	// 4. Filter list by category_id.
	filtered := getJSON(t, router, fmt.Sprintf("/accounts/%s/statements?category_id=%s", accountID, categoryID))
	require.Equal(t, http.StatusOK, filtered.Code)
	require.Len(t, extractListData(t, filtered.Body.Bytes()), 1, "filter by category_id must return exactly 1 statement")

	// 5. Filter list by tag_ids (any-of semantics).
	filtered2 := getJSON(t, router, fmt.Sprintf("/accounts/%s/statements?tag_ids=%s", accountID, tag1ID))
	require.Equal(t, http.StatusOK, filtered2.Code)
	assert.Len(t, extractListData(t, filtered2.Body.Bytes()), 1, "filter by tag_ids must return the matching statement")

	// 6. Try to delete the category — must return 409 (REQ-4).
	delReq := httptest.NewRequest(http.MethodDelete, "/categories/"+categoryID, nil)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)
	assert.Equal(t, http.StatusConflict, delW.Code, "delete category in use must return 409 ErrCategoryInUse; got body: %s", delW.Body.String())

	// 7. Sanity: statement still exists and balance is unchanged after the failed delete.
	balanceCheck := getJSON(t, router, fmt.Sprintf("/accounts/%s/statements", accountID))
	require.Equal(t, http.StatusOK, balanceCheck.Code)
	stmts := extractListData(t, balanceCheck.Body.Bytes())
	require.Len(t, stmts, 1)
	original := stmts[0].(map[string]interface{})
	assert.Equal(t, float64(2500), original["amount"])
	assert.Equal(t, float64(-2500), original["balance_after"])

	// 8. PATCH category — swap (debit→debit). Balance must remain unchanged (REQ-11 invariant).
	otherCatW := postJSON(t, router, "/categories", `{"name":"Restaurante","type":"debit"}`)
	require.Equal(t, http.StatusCreated, otherCatW.Code)
	otherCatID := extractData(t, otherCatW.Body.Bytes())["id"].(string)

	swapBody, _ := json.Marshal(map[string]string{"category_id": otherCatID})
	swapReq := httptest.NewRequest(http.MethodPatch, "/statements/"+stmtID+"/category", bytes.NewReader(swapBody))
	swapReq.Header.Set("Content-Type", "application/json")
	swapW := httptest.NewRecorder()
	router.ServeHTTP(swapW, swapReq)
	require.Equal(t, http.StatusOK, swapW.Code, "swap category: %s", swapW.Body.String())

	swapped := extractData(t, swapW.Body.Bytes())
	assert.Equal(t, otherCatID, swapped["category"].(map[string]interface{})["id"])
	assert.Equal(t, float64(2500), swapped["amount"], "amount must NEVER change in PATCH category")
	assert.Equal(t, float64(-2500), swapped["balance_after"], "balance_after must NEVER change in PATCH category")

	// 9. PATCH category with credit-typed category → 422.
	creditCatW := postJSON(t, router, "/categories", `{"name":"Salário","type":"credit"}`)
	require.Equal(t, http.StatusCreated, creditCatW.Code)
	creditCatID := extractData(t, creditCatW.Body.Bytes())["id"].(string)

	mismatchBody, _ := json.Marshal(map[string]string{"category_id": creditCatID})
	mismatchReq := httptest.NewRequest(http.MethodPatch, "/statements/"+stmtID+"/category", bytes.NewReader(mismatchBody))
	mismatchReq.Header.Set("Content-Type", "application/json")
	mismatchW := httptest.NewRecorder()
	router.ServeHTTP(mismatchW, mismatchReq)
	assert.Equal(t, http.StatusUnprocessableEntity, mismatchW.Code,
		"category type mismatch must return 422; got body: %s", mismatchW.Body.String())

	// 10. PATCH category with null clears it.
	clearReq := httptest.NewRequest(http.MethodPatch, "/statements/"+stmtID+"/category", bytes.NewBufferString(`{"category_id": null}`))
	clearReq.Header.Set("Content-Type", "application/json")
	clearW := httptest.NewRecorder()
	router.ServeHTTP(clearW, clearReq)
	require.Equal(t, http.StatusOK, clearW.Code, "clear category: %s", clearW.Body.String())
	cleared := extractData(t, clearW.Body.Bytes())
	assert.Nil(t, cleared["category"], "category must be null after clear")

	cleanupCategoriesTags(t)
}

// =============================================================================
// helpers (file-local — keep simple)
// =============================================================================

func postJSON(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getJSON(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func createTag(t *testing.T, router *gin.Engine, name string) string {
	t.Helper()
	w := postJSON(t, router, "/tags", fmt.Sprintf(`{"name":%q}`, name))
	require.Equal(t, http.StatusCreated, w.Code, "create tag: %s", w.Body.String())
	return extractData(t, w.Body.Bytes())["id"].(string)
}

// extractListData parses {"data": [...]} responses (list endpoints).
// extractData (in user_test.go) only handles object data — list endpoints carry a slice.
func extractListData(t *testing.T, body []byte) []interface{} {
	t.Helper()
	var envelope map[string]interface{}
	parseErr := json.Unmarshal(body, &envelope)
	require.NoError(t, parseErr)
	data, ok := envelope["data"].([]interface{})
	require.True(t, ok, "expected 'data' key with array value, got: %s", string(body))
	return data
}

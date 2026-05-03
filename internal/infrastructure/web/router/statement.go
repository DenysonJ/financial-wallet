package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/DenysonJ/financial-wallet/pkg/idempotency"
)

// RegisterStatementRoutes registers all Statement routes nested under accounts.
// idempotencyStore enforces the Idempotency-Key header on financial write endpoints
// (Create, Import, Reverse). Nil disables enforcement.
func RegisterStatementRoutes(rg *gin.RouterGroup, h *handler.StatementHandler, loader middleware.PermissionLoader, idempotencyStore idempotency.Store) {
	statements := rg.Group("/accounts/:id/statements")

	statements.POST("",
		middleware.RequirePermission(loader, "statement:write"),
		middleware.RequireIdempotencyKey(idempotencyStore),
		h.Create)
	statements.POST("/import",
		middleware.RequirePermission(loader, "statement:write"),
		middleware.RequireIdempotencyKey(idempotencyStore),
		h.Import)
	statements.GET("", middleware.RequirePermission(loader, "statement:read"), h.List)
	statements.GET("/:statement_id", middleware.RequirePermission(loader, "statement:read"), h.GetByID)
	statements.POST("/:statement_id/reverse",
		middleware.RequirePermission(loader, "statement:write"),
		middleware.RequireIdempotencyKey(idempotencyStore),
		h.Reverse)
}

// RegisterStatementMetadataRoutes registers the metadata-mutation endpoints
// (PATCH category, PUT tags). Top-level `/statements/:id/...` — not nested
// under accounts — because category/tags edits are pure metadata mutations
// that do not affect the account balance and have their own ownership flow
// (statement → account → user). Idempotency keys are NOT required because
// these operations are idempotent by definition
func RegisterStatementMetadataRoutes(rg *gin.RouterGroup, h *handler.StatementMetadataHandler, loader middleware.PermissionLoader) {
	rg.PATCH("/statements/:id/category",
		middleware.RequirePermission(loader, "statement:write"),
		h.UpdateCategory)
	rg.PUT("/statements/:id/tags",
		middleware.RequirePermission(loader, "statement:write"),
		h.ReplaceTags)
}

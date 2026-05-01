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

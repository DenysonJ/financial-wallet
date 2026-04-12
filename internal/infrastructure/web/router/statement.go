package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterStatementRoutes registers all Statement routes nested under accounts.
func RegisterStatementRoutes(rg *gin.RouterGroup, h *handler.StatementHandler, loader middleware.PermissionLoader) {
	statements := rg.Group("/accounts/:id/statements")

	statements.POST("", middleware.RequirePermission(loader, "statement:write"), h.Create)
	statements.GET("", middleware.RequirePermission(loader, "statement:read"), h.List)
	statements.GET("/:statement_id", middleware.RequirePermission(loader, "statement:read"), h.GetByID)
	statements.POST("/:statement_id/reverse", middleware.RequirePermission(loader, "statement:write"), h.Reverse)
}

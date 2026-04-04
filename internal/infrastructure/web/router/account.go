package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterAccountRoutes registra todas as rotas relacionadas a Account
func RegisterAccountRoutes(rg *gin.RouterGroup, h *handler.AccountHandler, loader middleware.PermissionLoader) {
	rg.POST("/accounts", middleware.RequirePermission(loader, "account:write"), h.Create)
	rg.GET("/accounts", middleware.RequirePermission(loader, "account:read"), h.List)
	rg.GET("/accounts/:id", middleware.RequirePermission(loader, "account:read"), h.GetByID)
	rg.PUT("/accounts/:id", middleware.RequirePermission(loader, "account:update"), h.Update)
	rg.DELETE("/accounts/:id", middleware.RequirePermission(loader, "account:delete"), h.Delete)
}

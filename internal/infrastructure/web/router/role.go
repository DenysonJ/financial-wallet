package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterRoleRoutes registra todas as rotas relacionadas a Role
func RegisterRoleRoutes(rg *gin.RouterGroup, h *handler.RoleHandler, loader middleware.PermissionLoader) {
	rg.POST("/roles", middleware.RequirePermission(loader, "role:write"), h.Create)
	rg.GET("/roles", middleware.RequirePermission(loader, "role:read"), h.List)
	rg.DELETE("/roles/:id", middleware.RequirePermission(loader, "role:delete"), h.Delete)
	rg.POST("/roles/:id/assign", middleware.RequirePermission(loader, "role:write"), h.AssignRole)
	rg.POST("/roles/:id/revoke", middleware.RequirePermission(loader, "role:write"), h.RevokeRole)
}

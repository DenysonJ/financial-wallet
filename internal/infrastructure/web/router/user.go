package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterUserRoutes registra todas as rotas relacionadas a User
func RegisterUserRoutes(rg *gin.RouterGroup, h *handler.UserHandler, loader middleware.PermissionLoader) {
	rg.POST("/users", middleware.RequirePermission(loader, "user:write"), h.Create)
	rg.GET("/users", middleware.RequirePermission(loader, "user:read"), h.List)
	rg.GET("/users/:id", middleware.RequirePermission(loader, "user:read"), h.GetByID)
	rg.PUT("/users/:id", middleware.RequirePermission(loader, "user:write"), h.Update)
	rg.DELETE("/users/:id", middleware.RequirePermission(loader, "user:delete"), h.Delete)
}

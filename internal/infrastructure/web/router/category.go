package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterCategoryRoutes registers the Category routes under /categories.
func RegisterCategoryRoutes(rg *gin.RouterGroup, h *handler.CategoryHandler, loader middleware.PermissionLoader) {
	rg.POST("/categories", middleware.RequirePermission(loader, "category:write"), h.Create)
	rg.GET("/categories", middleware.RequirePermission(loader, "category:read"), h.List)
	rg.PATCH("/categories/:id", middleware.RequirePermission(loader, "category:write"), h.Update)
	rg.DELETE("/categories/:id", middleware.RequirePermission(loader, "category:write"), h.Delete)
}

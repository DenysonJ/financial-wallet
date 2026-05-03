package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterTagRoutes registers the Tag routes under /tags.
func RegisterTagRoutes(rg *gin.RouterGroup, h *handler.TagHandler, loader middleware.PermissionLoader) {
	rg.POST("/tags", middleware.RequirePermission(loader, "tag:write"), h.Create)
	rg.GET("/tags", middleware.RequirePermission(loader, "tag:read"), h.List)
	rg.PATCH("/tags/:id", middleware.RequirePermission(loader, "tag:write"), h.Update)
	rg.DELETE("/tags/:id", middleware.RequirePermission(loader, "tag:write"), h.Delete)
}

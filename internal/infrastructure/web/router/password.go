package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
)

// RegisterSetPasswordRoute registra a rota de cadastro de senha.
func RegisterSetPasswordRoute(rg *gin.RouterGroup, h *handler.PasswordHandler, loader middleware.PermissionLoader) {
	rg.POST("/users/password", middleware.RequirePermission(loader, "user:write"), h.SetPassword)
}

// RegisterChangePasswordRoute registra a rota de alteração de senha.
func RegisterChangePasswordRoute(rg *gin.RouterGroup, h *handler.PasswordHandler, loader middleware.PermissionLoader) {
	rg.PUT("/users/password", middleware.RequirePermission(loader, "user:write"), h.ChangePassword)
}

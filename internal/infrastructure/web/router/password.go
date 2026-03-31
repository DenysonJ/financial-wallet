package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
)

// RegisterSetPasswordRoute registra a rota de cadastro de senha.
func RegisterSetPasswordRoute(rg *gin.RouterGroup, h *handler.PasswordHandler) {
	rg.POST("/users/password", h.SetPassword)
}

// RegisterChangePasswordRoute registra a rota de alteração de senha.
func RegisterChangePasswordRoute(rg *gin.RouterGroup, h *handler.PasswordHandler) {
	rg.PUT("/users/password", h.ChangePassword)
}

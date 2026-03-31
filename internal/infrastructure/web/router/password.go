package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
)

// RegisterPasswordRoutes registra as rotas de gerenciamento de senha.
func RegisterPasswordRoutes(rg *gin.RouterGroup, h *handler.PasswordHandler) {
	rg.POST("/users/password", h.SetPassword)
	rg.PUT("/users/password", h.ChangePassword)
}

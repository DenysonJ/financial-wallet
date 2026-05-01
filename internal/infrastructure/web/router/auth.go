package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
)

// RegisterAuthRoutes registra as rotas públicas de autenticação.
func RegisterAuthRoutes(rg *gin.RouterGroup, h *handler.AuthHandler) {
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
}

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
)

// RegisterAuthRoutes registra as rotas públicas de autenticação.
func RegisterAuthRoutes(r *gin.Engine, h *handler.AuthHandler) {
	auth := r.Group("/auth")
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/gin-gonic/gin"
)

// ContextKeyUserID is the Gin context key where the authenticated user ID is stored.
const ContextKeyUserID = "user_id"

// JWTAuth retorna um middleware que valida tokens JWT no header Authorization.
//
// Comportamento:
//   - Extrai o token do header "Authorization: Bearer <token>"
//   - Valida assinatura, expiração e tipo (deve ser "access")
//   - Salva user_id no contexto Gin para uso downstream
//   - Retorna 401 se o token é inválido, ausente ou expirado
func JWTAuth(tokenValidator interfaces.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, validateErr := tokenValidator.ValidateToken(tokenString)
		if validateErr != nil {
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		if claims.TokenType != interfaces.TokenTypeAccess {
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Next()
	}
}

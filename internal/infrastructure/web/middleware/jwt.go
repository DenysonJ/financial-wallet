package middleware

import (
	"net/http"
	"strings"

	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// JWTAuth retorna um middleware que valida tokens JWT no header Authorization.
//
// Comportamento:
//   - Extrai o token do header "Authorization: Bearer <token>"
//   - Valida assinatura, expiração e tipo (deve ser "access")
//   - Salva user_id no contexto Gin para uso downstream
//   - Retorna 401 se o token é inválido, ausente ou expirado
func JWTAuth(jwtService *jwt.Service) gin.HandlerFunc {
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

		claims, validateErr := jwtService.ValidateToken(tokenString)
		if validateErr != nil {
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		if claims.TokenType != jwt.TokenTypeAccess {
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

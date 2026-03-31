package interfaces

import "github.com/DenysonJ/financial-wallet/pkg/jwt"

// TokenService define o contrato para geração e validação de tokens JWT.
type TokenService interface {
	// GenerateAccessToken cria um access token de curta duração.
	GenerateAccessToken(userID string) (string, error)

	// GenerateRefreshToken cria um refresh token de longa duração.
	GenerateRefreshToken(userID string) (string, error)

	// ValidateToken valida um token e retorna as claims.
	ValidateToken(tokenString string) (*jwt.Claims, error)
}

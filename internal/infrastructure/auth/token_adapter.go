package auth

import (
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/jwt"
)

// JWTTokenAdapter adapts pkg/jwt.Service to the interfaces.TokenService contract.
// This adapter lives in the infrastructure layer, bridging the JWT package
// (infrastructure) with the use case interface (domain boundary).
type JWTTokenAdapter struct {
	service *jwt.Service
}

// NewJWTTokenAdapter creates a new adapter wrapping a JWT service.
func NewJWTTokenAdapter(service *jwt.Service) *JWTTokenAdapter {
	return &JWTTokenAdapter{service: service}
}

// GenerateAccessToken delegates to the underlying JWT service.
func (a *JWTTokenAdapter) GenerateAccessToken(userID string) (string, error) {
	return a.service.GenerateAccessToken(userID)
}

// GenerateRefreshToken delegates to the underlying JWT service.
func (a *JWTTokenAdapter) GenerateRefreshToken(userID string) (string, error) {
	return a.service.GenerateRefreshToken(userID)
}

// ValidateToken validates the token and maps jwt.Claims to interfaces.TokenClaims.
func (a *JWTTokenAdapter) ValidateToken(tokenString string) (*interfaces.TokenClaims, error) {
	claims, validateErr := a.service.ValidateToken(tokenString)
	if validateErr != nil {
		return nil, validateErr
	}

	return &interfaces.TokenClaims{
		UserID:    claims.UserID,
		TokenType: claims.TokenType,
	}, nil
}

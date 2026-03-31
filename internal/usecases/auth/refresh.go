package auth

import (
	"context"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
)

// RefreshUseCase implementa o caso de uso de refresh de token.
type RefreshUseCase struct {
	Token interfaces.TokenService
}

// NewRefreshUseCase cria uma nova instância do RefreshUseCase.
func NewRefreshUseCase(token interfaces.TokenService) *RefreshUseCase {
	return &RefreshUseCase{
		Token: token,
	}
}

// Execute valida um refresh token e gera um novo par de tokens.
//
// Fluxo:
//  1. Validar refresh token (assinatura, expiração)
//  2. Verificar que o tipo é "refresh"
//  3. Gerar novo access token e refresh token
func (uc *RefreshUseCase) Execute(ctx context.Context, input dto.RefreshInput) (*dto.RefreshOutput, error) {
	claims, validateErr := uc.Token.ValidateToken(input.RefreshToken)
	if validateErr != nil {
		return nil, userdomain.ErrInvalidCredentials
	}

	if claims.TokenType != interfaces.TokenTypeRefresh {
		return nil, userdomain.ErrInvalidCredentials
	}

	accessToken, accessErr := uc.Token.GenerateAccessToken(claims.UserID)
	if accessErr != nil {
		return nil, accessErr
	}

	refreshToken, refreshErr := uc.Token.GenerateRefreshToken(claims.UserID)
	if refreshErr != nil {
		return nil, refreshErr
	}

	return &dto.RefreshOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

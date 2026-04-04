package auth

import (
	"context"
	"testing"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/mocks/authuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func TestRefreshUseCase_Execute_Success(t *testing.T) {
	mockToken := authuci.NewMockTokenService(t)

	mockToken.On("ValidateToken", "valid-refresh").Return(&interfaces.TokenClaims{
		UserID:    "user-123",
		TokenType: interfaces.TokenTypeRefresh,
	}, nil)
	mockToken.On("GenerateAccessToken", "user-123").Return("new-access", nil)
	mockToken.On("GenerateRefreshToken", "user-123").Return("new-refresh", nil)

	uc := NewRefreshUseCase(mockToken)
	output, execErr := uc.Execute(context.Background(), dto.RefreshInput{
		RefreshToken: "valid-refresh",
	})

	assert.NoError(t, execErr)
	assert.Equal(t, "new-access", output.AccessToken)
	assert.Equal(t, "new-refresh", output.RefreshToken)
	mockToken.AssertExpectations(t)
}

func TestRefreshUseCase_Execute_InvalidToken(t *testing.T) {
	mockToken := authuci.NewMockTokenService(t)

	mockToken.On("ValidateToken", "invalid-token").Return(nil, jwt.ErrInvalidToken)

	uc := NewRefreshUseCase(mockToken)
	output, execErr := uc.Execute(context.Background(), dto.RefreshInput{
		RefreshToken: "invalid-token",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
}

func TestRefreshUseCase_Execute_WrongTokenType(t *testing.T) {
	mockToken := authuci.NewMockTokenService(t)

	mockToken.On("ValidateToken", "access-token").Return(&interfaces.TokenClaims{
		UserID:    "user-123",
		TokenType: interfaces.TokenTypeAccess,
	}, nil)

	uc := NewRefreshUseCase(mockToken)
	output, execErr := uc.Execute(context.Background(), dto.RefreshInput{
		RefreshToken: "access-token",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
	mockToken.AssertNotCalled(t, "GenerateAccessToken")
}

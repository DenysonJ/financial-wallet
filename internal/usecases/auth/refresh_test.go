package auth

import (
	"context"
	"errors"
	"testing"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/mocks/authuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRefreshUseCase_Execute(t *testing.T) {
	tests := []struct {
		name       string
		input      dto.RefreshInput
		setupMock  func(m *authuci.MockTokenService)
		wantErr    error
		wantErrMsg string
		wantAccess string
	}{
		{
			name:  "sucesso - gera novo par de tokens",
			input: dto.RefreshInput{RefreshToken: "valid-refresh"},
			setupMock: func(m *authuci.MockTokenService) {
				m.On("ValidateToken", "valid-refresh").Return(&interfaces.TokenClaims{
					UserID:    "user-123",
					TokenType: interfaces.TokenTypeRefresh,
				}, nil)
				m.On("GenerateAccessToken", "user-123").Return("new-access", nil)
				m.On("GenerateRefreshToken", "user-123").Return("new-refresh", nil)
			},
			wantAccess: "new-access",
		},
		{
			name:  "token inválido",
			input: dto.RefreshInput{RefreshToken: "invalid-token"},
			setupMock: func(m *authuci.MockTokenService) {
				m.On("ValidateToken", "invalid-token").Return(nil, jwt.ErrInvalidToken)
			},
			wantErr: userdomain.ErrInvalidCredentials,
		},
		{
			name:  "tipo de token errado (access ao invés de refresh)",
			input: dto.RefreshInput{RefreshToken: "access-token"},
			setupMock: func(m *authuci.MockTokenService) {
				m.On("ValidateToken", "access-token").Return(&interfaces.TokenClaims{
					UserID:    "user-123",
					TokenType: interfaces.TokenTypeAccess,
				}, nil)
			},
			wantErr: userdomain.ErrInvalidCredentials,
		},
		{
			name:  "erro ao gerar access token",
			input: dto.RefreshInput{RefreshToken: "valid-refresh"},
			setupMock: func(m *authuci.MockTokenService) {
				m.On("ValidateToken", "valid-refresh").Return(&interfaces.TokenClaims{
					UserID:    "user-123",
					TokenType: interfaces.TokenTypeRefresh,
				}, nil)
				m.On("GenerateAccessToken", "user-123").Return("", errors.New("signing key error"))
			},
			wantErrMsg: "signing key error",
		},
		{
			name:  "erro ao gerar refresh token",
			input: dto.RefreshInput{RefreshToken: "valid-refresh"},
			setupMock: func(m *authuci.MockTokenService) {
				m.On("ValidateToken", "valid-refresh").Return(&interfaces.TokenClaims{
					UserID:    "user-123",
					TokenType: interfaces.TokenTypeRefresh,
				}, nil)
				m.On("GenerateAccessToken", "user-123").Return("new-access", nil)
				m.On("GenerateRefreshToken", "user-123").Return("", errors.New("refresh signing error"))
			},
			wantErrMsg: "refresh signing error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockToken := authuci.NewMockTokenService(t)
			tt.setupMock(mockToken)

			uc := NewRefreshUseCase(mockToken)
			output, execErr := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				assert.Nil(t, output)
			} else if tt.wantErrMsg != "" {
				assert.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.wantErrMsg)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, execErr)
				assert.NotNil(t, output)
				assert.Equal(t, tt.wantAccess, output.AccessToken)
			}

			// Suppress unused mock expectation warnings for early-exit cases
			mock.AssertExpectationsForObjects(t, mockToken)
		})
	}
}

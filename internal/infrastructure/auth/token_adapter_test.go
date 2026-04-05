package auth

import (
	"testing"
	"time"

	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAdapter() *JWTTokenAdapter {
	service := jwt.NewService("test-secret-key-for-tests", 15*time.Minute, 168*time.Hour)
	return NewJWTTokenAdapter(service)
}

func TestJWTTokenAdapter_GenerateAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		wantErr      bool
		wantNonEmpty bool
	}{
		{
			name:         "sucesso",
			userID:       "user-123",
			wantNonEmpty: true,
		},
		{
			name:         "sucesso com user ID vazio",
			userID:       "",
			wantNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := newTestAdapter()

			token, genErr := adapter.GenerateAccessToken(tt.userID)

			if tt.wantErr {
				assert.Error(t, genErr)
			} else {
				assert.NoError(t, genErr)
			}
			if tt.wantNonEmpty {
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestJWTTokenAdapter_GenerateRefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		wantErr      bool
		wantNonEmpty bool
	}{
		{
			name:         "sucesso",
			userID:       "user-456",
			wantNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := newTestAdapter()

			token, genErr := adapter.GenerateRefreshToken(tt.userID)

			if tt.wantErr {
				assert.Error(t, genErr)
			} else {
				assert.NoError(t, genErr)
			}
			if tt.wantNonEmpty {
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestJWTTokenAdapter_ValidateToken(t *testing.T) {
	adapter := newTestAdapter()

	// Pre-generate tokens for test cases
	validAccess, accessErr := adapter.GenerateAccessToken("user-789")
	require.NoError(t, accessErr)

	validRefresh, refreshErr := adapter.GenerateRefreshToken("user-789")
	require.NoError(t, refreshErr)

	// Expired token: use a service with 0 TTL
	expiredService := jwt.NewService("test-secret-key-for-tests", -1*time.Second, -1*time.Second)
	expiredAdapter := NewJWTTokenAdapter(expiredService)
	expiredToken, expiredErr := expiredAdapter.GenerateAccessToken("user-expired")
	require.NoError(t, expiredErr)

	// Token signed with a different secret
	wrongKeyService := jwt.NewService("different-secret-key", 15*time.Minute, 168*time.Hour)
	wrongKeyAdapter := NewJWTTokenAdapter(wrongKeyService)
	wrongKeyToken, wrongKeyErr := wrongKeyAdapter.GenerateAccessToken("user-wrong")
	require.NoError(t, wrongKeyErr)

	tests := []struct {
		name          string
		token         string
		wantErr       error
		wantUserID    string
		wantTokenType string
	}{
		{
			name:          "sucesso — access token válido",
			token:         validAccess,
			wantUserID:    "user-789",
			wantTokenType: interfaces.TokenTypeAccess,
		},
		{
			name:          "sucesso — refresh token válido",
			token:         validRefresh,
			wantUserID:    "user-789",
			wantTokenType: interfaces.TokenTypeRefresh,
		},
		{
			name:    "token inválido — string malformada",
			token:   "not-a-jwt-token",
			wantErr: jwt.ErrInvalidToken,
		},
		{
			name:    "token inválido — string vazia",
			token:   "",
			wantErr: jwt.ErrInvalidToken,
		},
		{
			name:    "token expirado",
			token:   expiredToken,
			wantErr: jwt.ErrInvalidToken,
		},
		{
			name:    "token assinado com chave diferente",
			token:   wrongKeyToken,
			wantErr: jwt.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, validateErr := adapter.ValidateToken(tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, validateErr, tt.wantErr)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, validateErr)
				require.NotNil(t, claims)
				assert.Equal(t, tt.wantUserID, claims.UserID)
				assert.Equal(t, tt.wantTokenType, claims.TokenType)
			}
		})
	}
}

func TestJWTTokenAdapter_ValidateToken_MapsClaimsCorrectly(t *testing.T) {
	adapter := newTestAdapter()

	token, genErr := adapter.GenerateAccessToken("user-abc")
	require.NoError(t, genErr)

	claims, validateErr := adapter.ValidateToken(token)

	assert.NoError(t, validateErr)
	require.NotNil(t, claims)

	// Verify the adapter maps jwt.Claims → interfaces.TokenClaims correctly
	assert.Equal(t, "user-abc", claims.UserID)
	assert.Equal(t, interfaces.TokenTypeAccess, claims.TokenType)
	assert.IsType(t, &interfaces.TokenClaims{}, claims)
}

package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestService_GenerateAndValidateAccessToken(t *testing.T) {
	svc := NewService("test-secret-key-32chars!!", 15*time.Minute, 7*24*time.Hour)

	token, genErr := svc.GenerateAccessToken("user-123")
	assert.NoError(t, genErr)
	assert.NotEmpty(t, token)

	claims, valErr := svc.ValidateToken(token)
	assert.NoError(t, valErr)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, TokenTypeAccess, claims.TokenType)
}

func TestService_GenerateAndValidateRefreshToken(t *testing.T) {
	svc := NewService("test-secret-key-32chars!!", 15*time.Minute, 7*24*time.Hour)

	token, genErr := svc.GenerateRefreshToken("user-456")
	assert.NoError(t, genErr)
	assert.NotEmpty(t, token)

	claims, valErr := svc.ValidateToken(token)
	assert.NoError(t, valErr)
	assert.Equal(t, "user-456", claims.UserID)
	assert.Equal(t, TokenTypeRefresh, claims.TokenType)
}

func TestService_ValidateToken_Expired(t *testing.T) {
	svc := NewService("test-secret-key-32chars!!", 1*time.Millisecond, 1*time.Millisecond)

	token, genErr := svc.GenerateAccessToken("user-123")
	assert.NoError(t, genErr)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	claims, valErr := svc.ValidateToken(token)
	assert.ErrorIs(t, valErr, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestService_ValidateToken_InvalidSignature(t *testing.T) {
	svc1 := NewService("secret-key-one-32chars!!", 15*time.Minute, 7*24*time.Hour)
	svc2 := NewService("secret-key-two-32chars!!", 15*time.Minute, 7*24*time.Hour)

	token, genErr := svc1.GenerateAccessToken("user-123")
	assert.NoError(t, genErr)

	claims, valErr := svc2.ValidateToken(token)
	assert.ErrorIs(t, valErr, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestService_ValidateToken_Malformed(t *testing.T) {
	svc := NewService("test-secret-key-32chars!!", 15*time.Minute, 7*24*time.Hour)

	claims, valErr := svc.ValidateToken("not.a.valid.jwt")
	assert.ErrorIs(t, valErr, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestService_ValidateToken_EmptyString(t *testing.T) {
	svc := NewService("test-secret-key-32chars!!", 15*time.Minute, 7*24*time.Hour)

	claims, valErr := svc.ValidateToken("")
	assert.ErrorIs(t, valErr, ErrInvalidToken)
	assert.Nil(t, claims)
}

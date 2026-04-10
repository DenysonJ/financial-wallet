package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func FuzzValidateToken(f *testing.F) {
	svc := NewService("test-secret-for-fuzz-testing", 15*time.Minute, 168*time.Hour)

	// Generate a valid token to use as seed
	validToken, genErr := svc.GenerateAccessToken("user-123")
	if genErr != nil {
		f.Fatal(genErr)
	}

	f.Add(validToken)                                                                          // valid token
	f.Add("")                                                                                  // empty
	f.Add("not-a-jwt")                                                                         // garbage
	f.Add("a.b.c")                                                                             // three parts, invalid
	f.Add("eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoiMSJ9.")                                         // alg:none
	f.Add("eyJhbGciOiJIUzI1NiJ9.e30.ZRrHA1JJJW8opB1Qfp7QDm")                                   // minimal structure
	f.Add(string(make([]byte, 10000)))                                                         // very long
	f.Add("eyJ\x00.eyJ\x00.sig\x00")                                                           // null bytes
	f.Add("eyJhbGciOiJSUzI1NiJ9.eyJ1c2VyX2lkIjoiMSJ9.fakesig")                                 // RS256 header
	f.Add("..")                                                                                // empty parts
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c") // empty payload

	f.Fuzz(func(t *testing.T, tokenString string) {
		claims, validateErr := svc.ValidateToken(tokenString)

		// Must never panic
		if validateErr != nil {
			// All errors must be ErrInvalidToken
			assert.ErrorIs(t, validateErr, ErrInvalidToken)
			assert.Nil(t, claims)
			return
		}

		// If valid: claims must have required fields
		assert.NotNil(t, claims)
		assert.NotEmpty(t, claims.UserID)
		assert.True(t,
			claims.TokenType == TokenTypeAccess || claims.TokenType == TokenTypeRefresh,
			"token type must be access or refresh, got %q", claims.TokenType)
	})
}

func FuzzGenerateAndValidateToken(f *testing.F) {
	f.Add("user-123")                  // normal ID
	f.Add("")                          // empty
	f.Add("a")                         // single char
	f.Add(string(make([]byte, 10000))) // very long
	f.Add("user\x00null")              // null byte
	f.Add("ユーザー")                      // unicode
	f.Add("user with spaces")          // spaces
	f.Add("user@special#chars!")       // special chars

	svc := NewService("test-secret-for-fuzz-roundtrip", 15*time.Minute, 168*time.Hour)

	f.Fuzz(func(t *testing.T, userID string) {
		// Generate
		token, genErr := svc.GenerateAccessToken(userID)

		// Must never panic
		if genErr != nil {
			// Generation can fail for certain inputs; that's acceptable
			return
		}

		assert.NotEmpty(t, token)

		// Validate round-trip
		claims, validateErr := svc.ValidateToken(token)
		assert.NoError(t, validateErr)
		assert.NotNil(t, claims)

		if claims != nil {
			assert.Equal(t, userID, claims.UserID, "round-trip must preserve userID")
			assert.Equal(t, TokenTypeAccess, claims.TokenType)
		}
	})
}

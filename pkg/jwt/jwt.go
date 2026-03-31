package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when a token is malformed, expired, or has an invalid signature.
	ErrInvalidToken = errors.New("invalid token")

	// ErrInvalidTokenType is returned when the token type does not match the expected type.
	ErrInvalidTokenType = errors.New("invalid token type")
)

const (
	// TokenTypeAccess identifies an access token.
	TokenTypeAccess = "access"

	// TokenTypeRefresh identifies a refresh token.
	TokenTypeRefresh = "refresh"
)

// Claims represents the JWT claims payload.
type Claims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwtlib.RegisteredClaims
}

// Service handles JWT token generation and validation.
type Service struct {
	secretKey  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService creates a new JWT service.
func NewService(secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		secretKey:  []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// GenerateAccessToken creates a short-lived access token for the given user.
func (s *Service) GenerateAccessToken(userID string) (string, error) {
	return s.generateToken(userID, TokenTypeAccess, s.accessTTL)
}

// GenerateRefreshToken creates a long-lived refresh token for the given user.
func (s *Service) GenerateRefreshToken(userID string) (string, error) {
	return s.generateToken(userID, TokenTypeRefresh, s.refreshTTL)
}

// ValidateToken parses and validates a JWT token string.
// Returns the claims if valid, or ErrInvalidToken if not.
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, parseErr := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(token *jwtlib.Token) (any, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})
	if parseErr != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *Service) generateToken(userID, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwtlib.NewNumericDate(now),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signedToken, signErr := token.SignedString(s.secretKey)
	if signErr != nil {
		return "", signErr
	}

	return signedToken, nil
}

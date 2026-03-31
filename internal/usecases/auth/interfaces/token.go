package interfaces

// TokenClaims represents the validated claims from a token.
// Defined in the use case layer to avoid coupling to the JWT infrastructure package.
type TokenClaims struct {
	UserID    string
	TokenType string
}

// Token type constants for use case logic.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// TokenService define o contrato para geração e validação de tokens.
type TokenService interface {
	// GenerateAccessToken cria um access token de curta duração.
	GenerateAccessToken(userID string) (string, error)

	// GenerateRefreshToken cria um refresh token de longa duração.
	GenerateRefreshToken(userID string) (string, error)

	// ValidateToken valida um token e retorna as claims.
	ValidateToken(tokenString string) (*TokenClaims, error)
}

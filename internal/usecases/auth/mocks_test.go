package auth

import (
	"context"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository implements interfaces.UserRepository for tests.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email vo.Email) (*userdomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userdomain.User), args.Error(1)
}

// MockTokenService implements interfaces.TokenService for tests.
type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateAccessToken(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) GenerateRefreshToken(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateToken(tokenString string) (*interfaces.TokenClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TokenClaims), args.Error(1)
}

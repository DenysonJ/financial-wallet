package account

import (
	"context"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MockRepository - Mock do repositório de Account para testes unitários
// =============================================================================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, a *accountdomain.Account) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id uservo.ID) (*accountdomain.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*accountdomain.Account), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, filter accountdomain.ListFilter) (*accountdomain.ListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*accountdomain.ListResult), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, a *accountdomain.Account) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uservo.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

package role

import (
	"context"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MockRepository - Mock do repositorio de Role para testes unitarios
// =============================================================================

// MockRepository implementa a interface Repository para testes
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, r *roledomain.Role) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, filter roledomain.ListFilter) (*roledomain.ListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roledomain.ListResult), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id vo.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) FindByName(ctx context.Context, name string) (*roledomain.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roledomain.Role), args.Error(1)
}

func (m *MockRepository) FindByID(ctx context.Context, id vo.ID) (*roledomain.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roledomain.Role), args.Error(1)
}

func (m *MockRepository) AssignRole(ctx context.Context, userID vo.ID, roleID vo.ID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockRepository) RevokeRole(ctx context.Context, userID vo.ID, roleID vo.ID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockRepository) GetUserPermissions(ctx context.Context, userID vo.ID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRepository) GetUserRoles(ctx context.Context, userID vo.ID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

package middleware

import (
	"context"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MockPermissionLoader - Mock do PermissionLoader para testes unitários
// =============================================================================

// mockPermissionLoader implements PermissionLoader for tests.
type mockPermissionLoader struct {
	permissions []string
	err         error
}

func (m *mockPermissionLoader) GetPermissions(_ context.Context, _ string) ([]string, error) {
	return m.permissions, m.err
}

// =============================================================================
// MockPermissionRepo - Mock do PermissionRepository para testes unitários
// =============================================================================

// mockPermissionRepo implements PermissionRepository for tests.
type mockPermissionRepo struct {
	mock.Mock
}

func (m *mockPermissionRepo) GetUserPermissions(ctx context.Context, userID vo.ID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// =============================================================================
// MockCache - Mock da interface de Cache para testes unitários
// =============================================================================

// mockCache implements cache.Cache for tests.
type mockCache struct {
	mock.Mock
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

package statement

import (
	"context"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/mock"
)

// mockRepository is a hand-written mock for interfaces.Repository.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Create(ctx context.Context, stmt *stmtdomain.Statement, accountID vo.ID) (int64, error) {
	args := m.Called(ctx, stmt, accountID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockRepository) FindByID(ctx context.Context, id vo.ID) (*stmtdomain.Statement, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stmtdomain.Statement), args.Error(1)
}

func (m *mockRepository) List(ctx context.Context, filter stmtdomain.ListFilter) (*stmtdomain.ListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stmtdomain.ListResult), args.Error(1)
}

func (m *mockRepository) HasReversal(ctx context.Context, statementID vo.ID) (bool, error) {
	args := m.Called(ctx, statementID)
	return args.Bool(0), args.Error(1)
}

// mockAccountRepository is a hand-written mock for interfaces.AccountRepository.
type mockAccountRepository struct {
	mock.Mock
}

func (m *mockAccountRepository) FindByID(ctx context.Context, id vo.ID) (*accountdomain.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*accountdomain.Account), args.Error(1)
}

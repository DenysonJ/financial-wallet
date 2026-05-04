package statement

import (
	"context"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
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

func (m *mockRepository) CreateBatch(ctx context.Context, stmts []*stmtdomain.Statement, accountID vo.ID) (int64, error) {
	args := m.Called(ctx, stmts, accountID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockRepository) FindExternalIDs(ctx context.Context, accountID vo.ID, externalIDs []string) (map[string]bool, error) {
	args := m.Called(ctx, accountID, externalIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *mockRepository) UpdateCategory(ctx context.Context, statementID vo.ID, categoryID *vo.ID) error {
	args := m.Called(ctx, statementID, categoryID)
	return args.Error(0)
}

func (m *mockRepository) ReplaceTags(ctx context.Context, statementID vo.ID, tagIDs []vo.ID) error {
	args := m.Called(ctx, statementID, tagIDs)
	return args.Error(0)
}

func (m *mockRepository) CountByCategory(ctx context.Context, categoryID vo.ID) (int, error) {
	args := m.Called(ctx, categoryID)
	return args.Int(0), args.Error(1)
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

// mockCategoryReader satisfies interfaces.CategoryReader.
type mockCategoryReader struct {
	mock.Mock
}

func (m *mockCategoryReader) FindVisible(ctx context.Context, id, userID vo.ID) (*categorydomain.Category, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*categorydomain.Category), args.Error(1)
}

// mockTagReader satisfies interfaces.TagReader.
type mockTagReader struct {
	mock.Mock
}

func (m *mockTagReader) FindManyVisible(ctx context.Context, ids []vo.ID, userID vo.ID) ([]*tagdomain.Tag, error) {
	args := m.Called(ctx, ids, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*tagdomain.Tag), args.Error(1)
}

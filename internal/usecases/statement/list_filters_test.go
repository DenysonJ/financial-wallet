package statement

import (
	"context"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListUseCase_CategoryAndTagFilters(t *testing.T) {
	ownerID := vo.NewID()
	accountID := vo.NewID()
	now := time.Now()
	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Type: accountvo.TypeBankAccount,
		Active: true, Balance: 8000, CreatedAt: now, UpdatedAt: now,
	}

	categoryID := vo.NewID()
	tag1 := vo.NewID()
	tag2 := vo.NewID()

	tests := []struct {
		name        string
		input       dto.ListInput
		matchFilter func(stmtdomain.ListFilter) bool
		skipList    bool
		wantErr     error
	}{
		{
			name: "GIVEN category_id query param WHEN list THEN filter.CategoryID is propagated and TagIDs empty",
			input: dto.ListInput{
				CategoryID: categoryID.String(),
			},
			matchFilter: func(f stmtdomain.ListFilter) bool {
				return f.CategoryID != nil && *f.CategoryID == categoryID && len(f.TagIDs) == 0
			},
		},
		{
			name: "GIVEN tag_ids array (with duplicate) WHEN list THEN filter.TagIDs is dedup'd and propagated",
			input: dto.ListInput{
				TagIDs: []string{tag1.String(), tag2.String(), tag1.String()},
			},
			matchFilter: func(f stmtdomain.ListFilter) bool {
				return len(f.TagIDs) == 2
			},
		},
		{
			name: "GIVEN category + tags + type combined WHEN list THEN filter has all three",
			input: dto.ListInput{
				CategoryID: categoryID.String(),
				TagIDs:     []string{tag1.String()},
				Type:       "debit",
			},
			matchFilter: func(f stmtdomain.ListFilter) bool {
				return f.CategoryID != nil &&
					*f.CategoryID == categoryID &&
					len(f.TagIDs) == 1 &&
					f.Type != nil && f.Type.String() == "debit"
			},
		},
		{
			name: "GIVEN invalid category UUID WHEN list THEN returns ErrInvalidID (no DB call)",
			input: dto.ListInput{
				CategoryID: "not-a-uuid",
			},
			skipList: true,
			wantErr:  vo.ErrInvalidID,
		},
		{
			name: "GIVEN invalid tag UUID WHEN list THEN returns ErrInvalidID (no DB call)",
			input: dto.ListInput{
				TagIDs: []string{vo.NewID().String(), "not-a-uuid"},
			},
			skipList: true,
			wantErr:  vo.ErrInvalidID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			stmtRepo := &mockRepository{}
			accRepo := &mockAccountRepository{}

			tt.input.AccountID = accountID.String()
			tt.input.RequestingUserID = ownerID.String()

			accRepo.On("FindByID", mock.Anything, accountID).Return(activeAccount, nil)
			if !tt.skipList {
				stmtRepo.On("List", mock.Anything, mock.MatchedBy(tt.matchFilter)).
					Return(&stmtdomain.ListResult{
						Statements: []*stmtdomain.Statement{},
						Total:      0,
						Page:       1,
						Limit:      20,
					}, nil)
			}

			uc := NewListUseCase(stmtRepo, accRepo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				stmtRepo.AssertNotCalled(t, "List")
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
		})
	}
}

package statement

import (
	"context"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func makeDebitStatement(t *testing.T, accountID vo.ID, withCategory *vo.ID) *stmtdomain.Statement {
	t.Helper()
	now := time.Now()
	stmt := &stmtdomain.Statement{
		ID:           vo.NewID(),
		AccountID:    accountID,
		Type:         stmtvo.TypeDebit,
		Amount:       stmtvo.ParseAmount(2000),
		Description:  "purchase",
		BalanceAfter: 8000,
		PostedAt:     now,
		CreatedAt:    now,
		Tags:         []stmtdomain.TagRef{},
	}
	if withCategory != nil {
		c := *withCategory
		stmt.CategoryID = &c
	}
	return stmt
}

func ptrStr(s string) *string { return &s }

func TestUpdateCategoryUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	accountID := vo.NewID()
	otherOwner := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank",
		Type: accountvo.TypeBankAccount, Active: true, Balance: 8000,
		CreatedAt: now, UpdatedAt: now,
	}
	otherAccount := &accountdomain.Account{ID: accountID, UserID: otherOwner, Active: true}

	tests := []struct {
		name              string
		withCategoryOrig  bool                   // statement starts with a category
		findStmtErr       error                  // ErrStatementNotFound when nonexistent
		account           *accountdomain.Account // account returned by accRepo
		categoryID        *string                // input category_id (nil = clear)
		newCategoryFix    *categorydomain.Category
		findVisibleErr    error // when categoryID set
		updateErr         error
		skipFindStmt      bool
		skipFindAcc       bool
		skipFindCategory  bool
		skipUpdate        bool
		wantErr           error
		wantCategoryClear bool
		wantCategoryName  string
		wantSuccess       bool
	}{
		{
			name:             "GIVEN owned debit statement WHEN swap to debit category THEN persists and accounting fields are untouched",
			withCategoryOrig: true,
			account:          activeAccount,
			categoryID:       ptrStr(""), // overridden in test body with the resolved newCat ID
			newCategoryFix: &categorydomain.Category{
				UserID: &ownerID, Name: "Restaurante", Type: categoryvo.TypeDebit,
			},
			wantSuccess:      true,
			wantCategoryName: "Restaurante",
		},
		{
			name:              "GIVEN owned statement WHEN clear category (null) THEN persists null and emits null in output",
			withCategoryOrig:  true,
			account:           activeAccount,
			categoryID:        nil,
			skipFindCategory:  true,
			wantSuccess:       true,
			wantCategoryClear: true,
		},
		{
			name:       "GIVEN debit statement + credit category WHEN PATCH THEN returns ErrCategoryTypeMismatch (no UPDATE)",
			account:    activeAccount,
			categoryID: ptrStr(""), // overridden
			newCategoryFix: &categorydomain.Category{
				UserID: &ownerID, Name: "Salário", Type: categoryvo.TypeCredit,
			},
			skipUpdate: true,
			wantErr:    categorydomain.ErrCategoryTypeMismatch,
		},
		{
			name:             "GIVEN cross-user statement WHEN PATCH THEN returns ErrStatementNotFound (no oracle)",
			account:          otherAccount,
			categoryID:       nil,
			skipFindCategory: true,
			skipUpdate:       true,
			wantErr:          stmtdomain.ErrStatementNotFound,
		},
		{
			name:           "GIVEN invisible category WHEN PATCH THEN returns ErrCategoryNotVisible (422 not 404)",
			account:        activeAccount,
			categoryID:     ptrStr(""), // overridden
			findVisibleErr: categorydomain.ErrCategoryNotVisible,
			skipUpdate:     true,
			wantErr:        categorydomain.ErrCategoryNotVisible,
		},
		{
			name:             "GIVEN nonexistent statement WHEN PATCH THEN returns ErrStatementNotFound",
			findStmtErr:      stmtdomain.ErrStatementNotFound,
			skipFindAcc:      true,
			skipFindCategory: true,
			skipUpdate:       true,
			wantErr:          stmtdomain.ErrStatementNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			stmtRepo := &mockRepository{}
			accRepo := &mockAccountRepository{}
			catReader := &mockCategoryReader{}

			var oldCat *vo.ID
			if tt.withCategoryOrig {
				old := vo.NewID()
				oldCat = &old
			}
			stmt := makeDebitStatement(t, accountID, oldCat)

			// Snapshot accounting fields BEFORE the call.
			amountBefore := stmt.Amount.Int64()
			typeBefore := stmt.Type
			balanceBefore := stmt.BalanceAfter

			// Resolve the actual new category ID for inputs that need one.
			var inputCategoryID *string
			if tt.categoryID != nil {
				newCatID := vo.NewID()
				if tt.newCategoryFix != nil {
					tt.newCategoryFix.ID = newCatID
				}
				idStr := newCatID.String()
				inputCategoryID = &idStr
			}

			if tt.skipFindStmt || tt.findStmtErr != nil {
				if tt.findStmtErr != nil {
					stmtRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
						Return(nil, tt.findStmtErr)
				}
			} else {
				stmtRepo.On("FindByID", mock.Anything, stmt.ID).Return(stmt, nil)
			}
			if !tt.skipFindAcc {
				accRepo.On("FindByID", mock.Anything, accountID).Return(tt.account, nil)
			}
			if !tt.skipFindCategory && inputCategoryID != nil {
				catID, _ := vo.ParseID(*inputCategoryID)
				if tt.findVisibleErr != nil {
					catReader.On("FindVisible", mock.Anything, catID, ownerID).
						Return(nil, tt.findVisibleErr)
				} else {
					catReader.On("FindVisible", mock.Anything, catID, ownerID).
						Return(tt.newCategoryFix, nil)
				}
			}
			if !tt.skipUpdate {
				stmtRepo.On("UpdateCategory", mock.Anything, stmt.ID, mock.AnythingOfType("*vo.ID")).
					Return(tt.updateErr)
			}

			uc := NewUpdateCategoryUseCase(stmtRepo, accRepo, catReader)

			input := dto.UpdateCategoryInput{
				RequestingUserID: ownerID.String(),
				CategoryID:       inputCategoryID,
			}
			if tt.findStmtErr != nil {
				input.StatementID = vo.NewID().String()
			} else {
				input.StatementID = stmt.ID.String()
			}

			// Act
			out, execErr := uc.Execute(context.Background(), input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				if tt.skipUpdate {
					stmtRepo.AssertNotCalled(t, "UpdateCategory")
				}
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
			if tt.wantCategoryClear {
				assert.Nil(t, out.Category)
			} else if tt.wantSuccess {
				require.NotNil(t, out.Category)
				assert.Equal(t, tt.wantCategoryName, out.Category.Name)
			}
			// Accounting invariant — REQ-11.
			assert.Equal(t, amountBefore, out.Amount, "amount must NEVER change in PATCH category")
			assert.Equal(t, typeBefore.String(), out.Type, "type must NEVER change in PATCH category")
			assert.Equal(t, balanceBefore, out.BalanceAfter, "balance_after must NEVER change in PATCH category")
		})
	}
}

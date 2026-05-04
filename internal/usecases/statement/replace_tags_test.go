package statement

import (
	"context"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReplaceTagsUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	otherOwner := vo.NewID()
	accountID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Type: accountvo.TypeBankAccount,
		Active: true, Balance: 8000, CreatedAt: now, UpdatedAt: now,
	}
	otherAccount := &accountdomain.Account{ID: accountID, UserID: otherOwner, Active: true}

	// Build a fixed pool of tag IDs so subtests can reference them by index.
	tag1 := vo.NewID()
	tag2 := vo.NewID()

	tooMany := make([]string, 0, 11)
	for i := 0; i < 11; i++ {
		tooMany = append(tooMany, vo.NewID().String())
	}

	tests := []struct {
		name            string
		findStmtErr     error
		account         *accountdomain.Account
		inputTagIDs     []string
		visibleTags     []*tagdomain.Tag
		replaceErr      error
		skipFindStmt    bool
		skipFindAcc     bool
		skipFindManyVis bool
		skipReplace     bool
		wantErr         error
		wantTagsCount   int
		wantSuccess     bool
	}{
		{
			name:        "GIVEN owned statement WHEN PUT 2 visible tags (with one duplicate) THEN replaces and dedupes",
			account:     activeAccount,
			inputTagIDs: []string{tag1.String(), tag2.String(), tag1.String()},
			visibleTags: []*tagdomain.Tag{
				{ID: tag1, UserID: &ownerID, Name: "viagem"},
				{ID: tag2, Name: "recorrente"},
			},
			wantSuccess:   true,
			wantTagsCount: 2,
		},
		{
			name:            "GIVEN owned statement WHEN PUT empty array THEN clears tags (no FindManyVisible call)",
			account:         activeAccount,
			inputTagIDs:     []string{},
			skipFindManyVis: true,
			wantSuccess:     true,
			wantTagsCount:   0,
		},
		{
			name:            "GIVEN cross-user statement WHEN PUT THEN returns ErrStatementNotFound (no oracle)",
			account:         otherAccount,
			inputTagIDs:     []string{vo.NewID().String()},
			skipFindManyVis: true,
			skipReplace:     true,
			wantErr:         stmtdomain.ErrStatementNotFound,
		},
		{
			name:        "GIVEN invisible tag WHEN PUT THEN returns ErrTagNotVisible (2 requested, 1 visible)",
			account:     activeAccount,
			inputTagIDs: []string{tag1.String(), tag2.String()},
			visibleTags: []*tagdomain.Tag{{ID: tag1, UserID: &ownerID, Name: "x"}},
			skipReplace: true,
			wantErr:     tagdomain.ErrTagNotVisible,
		},
		{
			name:            "GIVEN >10 unique tags WHEN PUT THEN returns ErrTagLimitExceeded (no DB call to find tags)",
			account:         activeAccount,
			inputTagIDs:     tooMany,
			skipFindManyVis: true,
			skipReplace:     true,
			wantErr:         tagdomain.ErrTagLimitExceeded,
		},
		{
			name:            "GIVEN nonexistent statement WHEN PUT THEN returns ErrStatementNotFound",
			findStmtErr:     stmtdomain.ErrStatementNotFound,
			inputTagIDs:     []string{},
			skipFindAcc:     true,
			skipFindManyVis: true,
			skipReplace:     true,
			wantErr:         stmtdomain.ErrStatementNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			stmtRepo := &mockRepository{}
			accRepo := &mockAccountRepository{}
			tagReader := &mockTagReader{}

			stmt := makeDebitStatement(t, accountID, nil)
			amountBefore := stmt.Amount.Int64()
			balanceBefore := stmt.BalanceAfter

			if tt.findStmtErr != nil {
				stmtRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(nil, tt.findStmtErr)
			} else if !tt.skipFindStmt {
				stmtRepo.On("FindByID", mock.Anything, stmt.ID).Return(stmt, nil)
			}
			if !tt.skipFindAcc {
				accRepo.On("FindByID", mock.Anything, accountID).Return(tt.account, nil)
			}
			if !tt.skipFindManyVis {
				tagReader.On("FindManyVisible", mock.Anything, mock.Anything, ownerID).
					Return(tt.visibleTags, nil)
			}
			if !tt.skipReplace {
				stmtRepo.On("ReplaceTags", mock.Anything, stmt.ID, mock.MatchedBy(func(ids []vo.ID) bool {
					return len(ids) == tt.wantTagsCount
				})).Return(tt.replaceErr)
			}

			uc := NewReplaceTagsUseCase(stmtRepo, accRepo, tagReader)

			input := dto.ReplaceTagsInput{
				RequestingUserID: ownerID.String(),
				TagIDs:           tt.inputTagIDs,
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
				if tt.skipReplace {
					stmtRepo.AssertNotCalled(t, "ReplaceTags")
				}
				if tt.skipFindManyVis {
					tagReader.AssertNotCalled(t, "FindManyVisible")
				}
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
			assert.NotNil(t, out.Tags) // never nil
			require.Len(t, out.Tags, tt.wantTagsCount)
			// Accounting invariant — REQ-10.
			assert.Equal(t, amountBefore, out.Amount, "amount must NEVER change in PUT tags")
			assert.Equal(t, balanceBefore, out.BalanceAfter, "balance_after must NEVER change in PUT tags")
		})
	}
}

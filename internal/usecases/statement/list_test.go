package statement

import (
	"context"
	"errors"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	accountID := vo.NewID()
	stmtID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}

	listResult := &stmtdomain.ListResult{
		Statements: []*stmtdomain.Statement{
			{
				ID: stmtID, AccountID: accountID, Type: stmtvo.TypeCredit,
				Amount: stmtvo.ParseAmount(5000), Description: "Salary",
				BalanceAfter: 15000, CreatedAt: now,
			},
		},
		Total: 1, Page: 1, Limit: 20,
	}

	emptyResult := &stmtdomain.ListResult{
		Statements: []*stmtdomain.Statement{},
		Total:      0, Page: 1, Limit: 20,
	}

	tests := []struct {
		name            string
		input           dto.ListInput
		accountResult   *accountdomain.Account
		accountErr      error
		listResult      *stmtdomain.ListResult
		listErr         error
		wantErr         error
		wantErrMsg      string
		wantOutput      bool
		wantCount       int
		skipAccountCall bool
		skipListCall    bool
	}{
		{
			name: "given valid account when listing then returns statements",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Page: 1, Limit: 20,
			},
			accountResult: activeAccount,
			listResult:    listResult,
			wantOutput:    true,
			wantCount:     1,
		},
		{
			name: "given account with no statements when listing then returns empty",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
			},
			accountResult: activeAccount,
			listResult:    emptyResult,
			wantOutput:    true,
			wantCount:     0,
		},
		{
			name: "given type filter when listing then returns filtered results",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "credit",
			},
			accountResult: activeAccount,
			listResult:    listResult,
			wantOutput:    true,
			wantCount:     1,
		},
		{
			name: "given date range filter when listing then returns filtered results",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				DateFrom: "2026-01-01T00:00:00Z", DateTo: "2026-12-31T23:59:59Z",
			},
			accountResult: activeAccount,
			listResult:    listResult,
			wantOutput:    true,
			wantCount:     1,
		},
		{
			name: "given invalid account ID when listing then returns invalid ID",
			input: dto.ListInput{
				AccountID: "invalid", RequestingUserID: ownerID.String(),
			},
			wantErr:         vo.ErrInvalidID,
			skipAccountCall: true,
			skipListCall:    true,
		},
		{
			name: "given nonexistent account when listing then returns not found",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
			},
			accountErr:   accountdomain.ErrAccountNotFound,
			wantErr:      accountdomain.ErrAccountNotFound,
			skipListCall: true,
		},
		{
			name: "given other users account when listing then returns not found",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: otherUserID.String(),
			},
			accountResult: activeAccount,
			wantErr:       stmtdomain.ErrStatementNotFound,
			skipListCall:  true,
		},
		{
			name: "given invalid type filter when listing then returns invalid type",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "transfer",
			},
			accountResult: activeAccount,
			wantErr:       stmtvo.ErrInvalidStatementType,
			skipListCall:  true,
		},
		{
			name: "given invalid date_from when listing then returns parse error",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				DateFrom: "not-a-date",
			},
			accountResult: activeAccount,
			wantErrMsg:    "parsing time",
			skipListCall:  true,
		},
		{
			name: "given invalid date_to when listing then returns parse error",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				DateTo: "not-a-date",
			},
			accountResult: activeAccount,
			wantErrMsg:    "parsing time",
			skipListCall:  true,
		},
		{
			name: "given repo failure when listing then returns error",
			input: dto.ListInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
			},
			accountResult: activeAccount,
			listErr:       errors.New("db error"),
			wantErrMsg:    "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			mockAccRepo := &mockAccountRepository{}

			if !tt.skipAccountCall {
				mockAccRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.accountResult, tt.accountErr)
			}
			if !tt.skipListCall {
				mockRepo.On("List", mock.Anything, mock.AnythingOfType("statement.ListFilter")).
					Return(tt.listResult, tt.listErr)
			}

			uc := NewListUseCase(mockRepo, mockAccRepo)
			output, execErr := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				assert.Nil(t, output)
			} else if tt.wantErrMsg != "" {
				assert.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.wantErrMsg)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, execErr)
				assert.NotNil(t, output)
				assert.Len(t, output.Data, tt.wantCount)
				assert.NotZero(t, output.Pagination.Limit)
			}

			if tt.skipListCall {
				mockRepo.AssertNotCalled(t, "List")
			}
			if tt.skipAccountCall {
				mockAccRepo.AssertNotCalled(t, "FindByID")
			}
		})
	}
}

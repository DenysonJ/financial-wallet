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

func TestGetUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	accountID := vo.NewID()
	otherAccountID := vo.NewID()
	statementID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}

	validStatement := &stmtdomain.Statement{
		ID: statementID, AccountID: accountID, Type: stmtvo.TypeCredit,
		Amount: stmtvo.ParseAmount(5000), Description: "Salary",
		BalanceAfter: 15000, CreatedAt: now,
	}
	statementOtherAccount := &stmtdomain.Statement{
		ID: statementID, AccountID: otherAccountID, Type: stmtvo.TypeCredit,
		Amount: stmtvo.ParseAmount(5000), BalanceAfter: 5000, CreatedAt: now,
	}

	tests := []struct {
		name            string
		input           dto.GetInput
		findResult      *stmtdomain.Statement
		findErr         error
		accountResult   *accountdomain.Account
		accountErr      error
		wantErr         error
		wantErrMsg      string
		wantOutput      bool
		skipFindCall    bool
		skipAccountCall bool
	}{
		{
			name: "sucesso com ownership check",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			findResult:    validStatement,
			accountResult: activeAccount,
			wantOutput:    true,
		},
		{
			name: "sucesso sem ownership check (admin)",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
			},
			findResult:    validStatement,
			accountResult: activeAccount,
			wantOutput:    true,
		},
		{
			name: "statement ID inválido",
			input: dto.GetInput{
				ID: "invalid", AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			wantErr:         vo.ErrInvalidID,
			skipFindCall:    true,
			skipAccountCall: true,
		},
		{
			name: "account ID inválido",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: "invalid",
				RequestingUserID: ownerID.String(),
			},
			wantErr:         vo.ErrInvalidID,
			skipFindCall:    true,
			skipAccountCall: true,
		},
		{
			name: "statement não encontrado",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			findErr:         stmtdomain.ErrStatementNotFound,
			wantErr:         stmtdomain.ErrStatementNotFound,
			skipAccountCall: true,
		},
		{
			name: "statement pertence a outra account",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			findResult:      statementOtherAccount,
			wantErr:         stmtdomain.ErrStatementNotFound,
			skipAccountCall: true,
		},
		{
			name: "não é dono da account",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: otherUserID.String(),
			},
			findResult:    validStatement,
			accountResult: activeAccount,
			wantErr:       stmtdomain.ErrStatementNotFound,
		},
		{
			name: "account não encontrada",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			findResult: validStatement,
			accountErr: accountdomain.ErrAccountNotFound,
			wantErr:    accountdomain.ErrAccountNotFound,
		},
		{
			name: "erro do repositório",
			input: dto.GetInput{
				ID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			findErr:         errors.New("db error"),
			wantErrMsg:      "db error",
			skipAccountCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			mockAccRepo := &mockAccountRepository{}

			if !tt.skipFindCall {
				mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.findResult, tt.findErr)
			}
			if !tt.skipAccountCall {
				mockAccRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.accountResult, tt.accountErr)
			}

			uc := NewGetUseCase(mockRepo, mockAccRepo)
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
				assert.Equal(t, statementID.String(), output.ID)
				assert.Equal(t, accountID.String(), output.AccountID)
				assert.Equal(t, "credit", output.Type)
				assert.Equal(t, int64(5000), output.Amount)
				assert.Equal(t, "Salary", output.Description)
				assert.Equal(t, int64(15000), output.BalanceAfter)
			}

			if tt.skipFindCall {
				mockRepo.AssertNotCalled(t, "FindByID")
			}
			if tt.skipAccountCall {
				mockAccRepo.AssertNotCalled(t, "FindByID")
			}
		})
	}
}

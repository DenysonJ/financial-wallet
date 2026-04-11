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

func TestReverseUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	accountID := vo.NewID()
	statementID := vo.NewID()
	otherAccountID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}
	inactiveAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: false, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}

	originalCredit := &stmtdomain.Statement{
		ID: statementID, AccountID: accountID, Type: stmtvo.TypeCredit,
		Amount: stmtvo.ParseAmount(5000), Description: "Salary",
		BalanceAfter: 15000, CreatedAt: now,
	}
	statementOtherAccount := &stmtdomain.Statement{
		ID: statementID, AccountID: otherAccountID, Type: stmtvo.TypeCredit,
		Amount: stmtvo.ParseAmount(5000), Description: "Other",
		BalanceAfter: 5000, CreatedAt: now,
	}

	tests := []struct {
		name              string
		input             dto.ReverseInput
		accountResult     *accountdomain.Account
		accountErr        error
		findResult        *stmtdomain.Statement
		findErr           error
		hasReversal       bool
		hasReversalErr    error
		repoCreateErr     error
		wantErr           error
		wantErrMsg        string
		wantOutput        bool
		skipAccountCall   bool
		skipFindCall      bool
		skipReversalCheck bool
		skipCreateCall    bool
	}{
		{
			name: "sucesso - reversal de credit gera debit",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(), Description: "Reversal",
			},
			accountResult: activeAccount,
			findResult:    originalCredit,
			hasReversal:   false,
			wantOutput:    true,
		},
		{
			name: "statement ID inválido",
			input: dto.ReverseInput{
				StatementID: "invalid", AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			wantErr:           vo.ErrInvalidID,
			skipAccountCall:   true,
			skipFindCall:      true,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "account ID inválido",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: "invalid",
				RequestingUserID: ownerID.String(),
			},
			wantErr:           vo.ErrInvalidID,
			skipAccountCall:   true,
			skipFindCall:      true,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "account não encontrada",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			accountErr:        accountdomain.ErrAccountNotFound,
			wantErr:           accountdomain.ErrAccountNotFound,
			skipFindCall:      true,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "não é dono da account",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: otherUserID.String(),
			},
			accountResult:     activeAccount,
			wantErr:           stmtdomain.ErrStatementNotFound,
			skipFindCall:      true,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "account inativa",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			accountResult:     inactiveAccount,
			wantErr:           stmtdomain.ErrAccountNotActive,
			skipFindCall:      true,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "statement não encontrado",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			accountResult:     activeAccount,
			findErr:           stmtdomain.ErrStatementNotFound,
			wantErr:           stmtdomain.ErrStatementNotFound,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "statement pertence a outra account",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			accountResult:     activeAccount,
			findResult:        statementOtherAccount,
			wantErr:           stmtdomain.ErrStatementNotFound,
			skipReversalCheck: true,
			skipCreateCall:    true,
		},
		{
			name: "já foi revertido",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			accountResult:  activeAccount,
			findResult:     originalCredit,
			hasReversal:    true,
			wantErr:        stmtdomain.ErrAlreadyReversed,
			skipCreateCall: true,
		},
		{
			name: "erro no repositório ao criar",
			input: dto.ReverseInput{
				StatementID: statementID.String(), AccountID: accountID.String(),
				RequestingUserID: ownerID.String(),
			},
			accountResult: activeAccount,
			findResult:    originalCredit,
			hasReversal:   false,
			repoCreateErr: errors.New("db error"),
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
			if !tt.skipFindCall {
				mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.findResult, tt.findErr)
			}
			if !tt.skipReversalCheck {
				mockRepo.On("HasReversal", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.hasReversal, tt.hasReversalErr)
			}
			if !tt.skipCreateCall {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), mock.AnythingOfType("vo.ID")).
					Return(tt.repoCreateErr)
			}

			uc := NewReverseUseCase(mockRepo, mockAccRepo)
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
				assert.NotEmpty(t, output.ID)
				assert.Equal(t, accountID.String(), output.AccountID)
				assert.Equal(t, "debit", output.Type) // opposite of credit
				assert.Equal(t, int64(5000), output.Amount)
				assert.NotNil(t, output.ReferenceID)
				assert.Equal(t, statementID.String(), *output.ReferenceID)
			}

			if tt.skipCreateCall {
				mockRepo.AssertNotCalled(t, "Create")
			}
		})
	}
}

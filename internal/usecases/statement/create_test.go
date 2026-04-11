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

func TestCreateUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	accountID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}
	inactiveAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: false, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name            string
		input           dto.CreateInput
		accountResult   *accountdomain.Account
		accountErr      error
		repoErr         error
		wantErr         error
		wantErrMsg      string
		wantOutput      bool
		skipAccountCall bool
		skipRepoCall    bool
	}{
		{
			name: "sucesso credit",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "credit", Amount: 5000, Description: "Salary",
			},
			accountResult: activeAccount,
			wantOutput:    true,
		},
		{
			name: "sucesso debit",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "debit", Amount: 2000, Description: "Purchase",
			},
			accountResult: activeAccount,
			wantOutput:    true,
		},
		{
			name: "account ID inválido",
			input: dto.CreateInput{
				AccountID: "invalid", RequestingUserID: ownerID.String(),
				Type: "credit", Amount: 1000,
			},
			wantErr:         vo.ErrInvalidID,
			skipAccountCall: true,
			skipRepoCall:    true,
		},
		{
			name: "account não encontrada",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "credit", Amount: 1000,
			},
			accountErr:   accountdomain.ErrAccountNotFound,
			wantErr:      accountdomain.ErrAccountNotFound,
			skipRepoCall: true,
		},
		{
			name: "não é dono da account",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: otherUserID.String(),
				Type: "credit", Amount: 1000,
			},
			accountResult: activeAccount,
			wantErr:       stmtdomain.ErrStatementNotFound,
			skipRepoCall:  true,
		},
		{
			name: "account inativa",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "credit", Amount: 1000,
			},
			accountResult: inactiveAccount,
			wantErr:       stmtdomain.ErrAccountNotActive,
			skipRepoCall:  true,
		},
		{
			name: "tipo inválido",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "transfer", Amount: 1000,
			},
			accountResult: activeAccount,
			wantErr:       stmtvo.ErrInvalidStatementType,
			skipRepoCall:  true,
		},
		{
			name: "amount zero",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "credit", Amount: 0,
			},
			accountResult: activeAccount,
			wantErr:       stmtvo.ErrInvalidAmount,
			skipRepoCall:  true,
		},
		{
			name: "amount negativo",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "credit", Amount: -100,
			},
			accountResult: activeAccount,
			wantErr:       stmtvo.ErrInvalidAmount,
			skipRepoCall:  true,
		},
		{
			name: "erro do repositório",
			input: dto.CreateInput{
				AccountID: accountID.String(), RequestingUserID: ownerID.String(),
				Type: "debit", Amount: 5000, Description: "Payment",
			},
			accountResult: activeAccount,
			repoErr:       errors.New("database error"),
			wantErrMsg:    "database error",
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
			if !tt.skipRepoCall {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), mock.AnythingOfType("vo.ID")).
					Return(tt.repoErr)
			}

			uc := NewCreateUseCase(mockRepo, mockAccRepo)
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
				assert.Equal(t, tt.input.Type, output.Type)
				assert.Equal(t, tt.input.Amount, output.Amount)
				assert.Equal(t, tt.input.Description, output.Description)
				assert.NotEmpty(t, output.CreatedAt)
			}

			if tt.skipRepoCall {
				mockRepo.AssertNotCalled(t, "Create")
			}
			if tt.skipAccountCall {
				mockAccRepo.AssertNotCalled(t, "FindByID")
			}
		})
	}
}

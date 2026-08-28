package account

import (
	"context"
	"errors"
	"testing"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/accountuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func creditLimitPtr(v int64) *int64 { return &v }

func TestCreateUseCase_Execute(t *testing.T) {
	tests := []struct {
		name         string
		input        dto.CreateInput
		repoErr      error
		wantErr      error
		wantErrMsg   string
		wantOutput   bool
		skipRepoCall bool
	}{
		{
			name: "sucesso",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Nubank", Type: "bank_account", Description: "Conta corrente",
			},
			wantOutput: true,
		},
		{
			name: "user ID inválido",
			input: dto.CreateInput{
				UserID: "invalid-id", Name: "Nubank", Type: "bank_account",
			},
			wantErr:      vo.ErrInvalidID,
			skipRepoCall: true,
		},
		{
			name: "tipo inválido",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Nubank", Type: "savings",
			},
			wantErr:      accountvo.ErrInvalidAccountType,
			skipRepoCall: true,
		},
		{
			name: "erro do repositório",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Nubank", Type: "bank_account",
			},
			repoErr:    errors.New("database connection failed"),
			wantErrMsg: "database connection failed",
		},
		{
			name: "GIVEN a credit card WHEN created with a valid credit limit THEN persists it (RN-05)",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Cartão Nubank", Type: "credit_card", CreditLimit: creditLimitPtr(500000),
			},
			wantOutput: true,
		},
		{
			name: "GIVEN a credit card WHEN created without a credit limit THEN fails as required (RN-01)",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Cartão Nubank", Type: "credit_card",
			},
			wantErr:      accountdomain.ErrCreditLimitRequired,
			skipRepoCall: true,
		},
		{
			name: "GIVEN a bank account WHEN created with a credit limit THEN fails as not allowed (RN-02)",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Nubank", Type: "bank_account", CreditLimit: creditLimitPtr(100000),
			},
			wantErr:      accountdomain.ErrCreditLimitNotAllowed,
			skipRepoCall: true,
		},
		{
			name: "GIVEN a cash account WHEN created with a zero credit limit THEN fails by type, not by value (RN-02)",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Carteira", Type: "cash", CreditLimit: creditLimitPtr(0),
			},
			wantErr:      accountdomain.ErrCreditLimitNotAllowed,
			skipRepoCall: true,
		},
		{
			name: "GIVEN a credit card WHEN created with a zero credit limit THEN fails as invalid (RN-03)",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Cartão", Type: "credit_card", CreditLimit: creditLimitPtr(0),
			},
			wantErr:      accountvo.ErrInvalidCreditLimit,
			skipRepoCall: true,
		},
		{
			name: "GIVEN a credit card WHEN created above the credit limit ceiling THEN fails as invalid (RN-03)",
			input: dto.CreateInput{
				UserID: vo.NewID().String(), Name: "Cartão", Type: "credit_card", CreditLimit: creditLimitPtr(accountvo.MaxCreditLimit + 1),
			},
			wantErr:      accountvo.ErrInvalidCreditLimit,
			skipRepoCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := accountuci.NewMockRepository(t)
			if !tt.skipRepoCall {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return(tt.repoErr)
			}

			uc := NewCreateUseCase(mockRepo)
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
				assert.NotEmpty(t, output.CreatedAt)
			}

			if tt.skipRepoCall {
				mockRepo.AssertNotCalled(t, "Create")
			} else {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

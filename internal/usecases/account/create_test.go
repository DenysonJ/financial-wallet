package account

import (
	"context"
	"errors"
	"testing"

	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/accountuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

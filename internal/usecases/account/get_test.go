package account

import (
	"context"
	"errors"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/accountuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetUseCase_Execute(t *testing.T) {
	validID := vo.NewID()
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	now := time.Now()

	validAccount := &accountdomain.Account{
		ID: validID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Description: "Conta corrente", Active: true, CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name         string
		input        dto.GetInput
		repoResult   *accountdomain.Account
		repoErr      error
		wantErr      error
		wantErrMsg   string
		wantOutput   bool
		skipRepoCall bool
	}{
		{
			name:       "sucesso sem ownership check",
			input:      dto.GetInput{ID: validID.String()},
			repoResult: validAccount,
			wantOutput: true,
		},
		{
			name:       "sucesso com ownership check - dono",
			input:      dto.GetInput{ID: validID.String(), RequestingUserID: ownerID.String()},
			repoResult: validAccount,
			wantOutput: true,
		},
		{
			name:       "forbidden - não é dono (retorna not found)",
			input:      dto.GetInput{ID: validID.String(), RequestingUserID: otherUserID.String()},
			repoResult: validAccount,
			wantErr:    accountdomain.ErrAccountNotFound,
		},
		{
			name:       "ownership skip quando RequestingUserID vazio (admin/service-key)",
			input:      dto.GetInput{ID: validID.String(), RequestingUserID: ""},
			repoResult: validAccount,
			wantOutput: true,
		},
		{
			name:    "não encontrado",
			input:   dto.GetInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456"},
			repoErr: accountdomain.ErrAccountNotFound,
			wantErr: accountdomain.ErrAccountNotFound,
		},
		{
			name:         "ID inválido",
			input:        dto.GetInput{ID: "invalid-id"},
			skipRepoCall: true,
			wantErr:      vo.ErrInvalidID,
		},
		{
			name:       "erro do repositório",
			input:      dto.GetInput{ID: validID.String()},
			repoErr:    errors.New("db error"),
			wantErrMsg: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := accountuci.NewMockRepository(t)
			if !tt.skipRepoCall {
				mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.repoResult, tt.repoErr)
			}

			uc := NewGetUseCase(mockRepo)
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
				assert.Equal(t, validID.String(), output.ID)
			}

			if tt.skipRepoCall {
				mockRepo.AssertNotCalled(t, "FindByID")
			} else {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

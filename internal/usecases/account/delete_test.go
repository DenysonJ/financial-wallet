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

func TestDeleteUseCase_Execute(t *testing.T) {
	validID := vo.NewID()
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	now := time.Now()

	ownedAccount := &accountdomain.Account{
		ID: validID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name           string
		input          dto.DeleteInput
		findResult     *accountdomain.Account
		findErr        error
		deleteErr      error
		wantErr        error
		wantErrMsg     string
		wantOutput     bool
		skipFindCall   bool
		skipDeleteCall bool
	}{
		{
			name:       "sucesso sem ownership check",
			input:      dto.DeleteInput{ID: validID.String()},
			wantOutput: true,
		},
		{
			name:       "sucesso com ownership check - dono",
			input:      dto.DeleteInput{ID: validID.String(), RequestingUserID: ownerID.String()},
			findResult: ownedAccount,
			wantOutput: true,
		},
		{
			name:         "não encontrado",
			input:        dto.DeleteInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456"},
			deleteErr:    accountdomain.ErrAccountNotFound,
			wantErr:      accountdomain.ErrAccountNotFound,
			skipFindCall: true,
		},
		{
			name:           "ID inválido",
			input:          dto.DeleteInput{ID: "invalid"},
			wantErr:        vo.ErrInvalidID,
			skipFindCall:   true,
			skipDeleteCall: true,
		},
		{
			name:       "erro do repositório",
			input:      dto.DeleteInput{ID: validID.String()},
			deleteErr:  errors.New("db error"),
			wantErrMsg: "db error",
		},
		{
			name:           "ownership check - não é dono (retorna not found)",
			input:          dto.DeleteInput{ID: validID.String(), RequestingUserID: otherUserID.String()},
			findResult:     ownedAccount,
			wantErr:        accountdomain.ErrAccountNotFound,
			skipDeleteCall: true,
		},
		{
			name:         "double delete - já deletado retorna not found",
			input:        dto.DeleteInput{ID: validID.String()},
			deleteErr:    accountdomain.ErrAccountNotFound,
			wantErr:      accountdomain.ErrAccountNotFound,
			skipFindCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := accountuci.NewMockRepository(t)

			// Setup FindByID mock (only for ownership check cases)
			if !tt.skipFindCall && tt.input.RequestingUserID != "" {
				mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.findResult, tt.findErr)
			}

			// Setup Delete mock
			if !tt.skipDeleteCall {
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.deleteErr)
			}

			uc := NewDeleteUseCase(mockRepo)
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
				assert.NotEmpty(t, output.DeletedAt)
			}

			if tt.skipDeleteCall {
				mockRepo.AssertNotCalled(t, "Delete")
			}
		})
	}
}

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

func ptrStr(s string) *string { return &s }

func newExistingAccount(id, ownerID vo.ID) *accountdomain.Account {
	return &accountdomain.Account{
		ID: id, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Description: "Original", Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func TestUpdateUseCase_Execute(t *testing.T) {
	validID := vo.NewID()
	ownerID := vo.NewID()
	otherUserID := vo.NewID()

	tests := []struct {
		name           string
		input          dto.UpdateInput
		buildAccount   func() *accountdomain.Account
		findErr        error
		updateErr      error
		wantErr        error
		wantErrMsg     string
		wantName       string
		wantDesc       string
		skipFindCall   bool
		skipUpdateCall bool
	}{
		{
			name:         "sucesso - atualiza nome e descrição",
			input:        dto.UpdateInput{ID: validID.String(), Name: ptrStr("Nubank Ultravioleta"), Description: ptrStr("Premium")},
			buildAccount: func() *accountdomain.Account { return newExistingAccount(validID, ownerID) },
			wantName:     "Nubank Ultravioleta",
			wantDesc:     "Premium",
		},
		{
			name:         "sucesso - atualiza apenas nome",
			input:        dto.UpdateInput{ID: validID.String(), Name: ptrStr("New Name")},
			buildAccount: func() *accountdomain.Account { return newExistingAccount(validID, ownerID) },
			wantName:     "New Name",
			wantDesc:     "Original",
		},
		{
			name:         "sucesso - no-op (ambos campos nil)",
			input:        dto.UpdateInput{ID: validID.String()},
			buildAccount: func() *accountdomain.Account { return newExistingAccount(validID, ownerID) },
			wantName:     "Nubank",
			wantDesc:     "Original",
		},
		{
			name:           "não encontrado",
			input:          dto.UpdateInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456", Name: ptrStr("X")},
			findErr:        accountdomain.ErrAccountNotFound,
			wantErr:        accountdomain.ErrAccountNotFound,
			skipUpdateCall: true,
		},
		{
			name:           "ID inválido",
			input:          dto.UpdateInput{ID: "invalid", Name: ptrStr("X")},
			wantErr:        vo.ErrInvalidID,
			skipFindCall:   true,
			skipUpdateCall: true,
		},
		{
			name:         "erro do repositório no update",
			input:        dto.UpdateInput{ID: validID.String(), Name: ptrStr("X")},
			buildAccount: func() *accountdomain.Account { return newExistingAccount(validID, ownerID) },
			updateErr:    errors.New("db error"),
			wantErrMsg:   "db error",
		},
		{
			name:           "ownership check - não é dono (retorna not found)",
			input:          dto.UpdateInput{ID: validID.String(), RequestingUserID: otherUserID.String(), Name: ptrStr("Hacked")},
			buildAccount:   func() *accountdomain.Account { return newExistingAccount(validID, ownerID) },
			wantErr:        accountdomain.ErrAccountNotFound,
			skipUpdateCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := accountuci.NewMockRepository(t)

			if !tt.skipFindCall {
				var findResult *accountdomain.Account
				if tt.buildAccount != nil {
					findResult = tt.buildAccount()
				}
				mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(findResult, tt.findErr)
			}
			if !tt.skipUpdateCall && tt.updateErr != nil {
				mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).
					Return(tt.updateErr)
			} else if !tt.skipUpdateCall && tt.findErr == nil && tt.wantErr == nil && tt.wantErrMsg == "" {
				mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).
					Return(nil)
			}

			uc := NewUpdateUseCase(mockRepo)
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
				assert.Equal(t, tt.wantName, output.Name)
				assert.Equal(t, tt.wantDesc, output.Description)
			}

			if tt.skipFindCall {
				mockRepo.AssertNotCalled(t, "FindByID")
			}
			if tt.skipUpdateCall {
				mockRepo.AssertNotCalled(t, "Update")
			}
		})
	}
}

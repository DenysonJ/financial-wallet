package account

import (
	"context"
	"errors"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListUseCase_Execute(t *testing.T) {
	userID := uservo.NewID()
	now := time.Now()

	twoAccounts := &accountdomain.ListResult{
		Accounts: []*accountdomain.Account{
			{ID: uservo.NewID(), UserID: userID, Name: "Nubank", Type: accountvo.TypeBankAccount, Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: uservo.NewID(), UserID: userID, Name: "Cartão Inter", Type: accountvo.TypeCreditCard, Active: true, CreatedAt: now, UpdatedAt: now},
		},
		Total: 2, Page: 1, Limit: 20,
	}

	oneAccount := &accountdomain.ListResult{
		Accounts: []*accountdomain.Account{
			{ID: uservo.NewID(), UserID: userID, Name: "Nubank", Type: accountvo.TypeBankAccount, Active: true, CreatedAt: now, UpdatedAt: now},
		},
		Total: 1, Page: 1, Limit: 20,
	}

	emptyResult := &accountdomain.ListResult{
		Accounts: []*accountdomain.Account{}, Total: 0, Page: 1, Limit: 20,
	}

	tests := []struct {
		name       string
		input      dto.ListInput
		repoResult *accountdomain.ListResult
		repoErr    error
		wantErr    bool
		errSubstr  string
		wantTotal  int
		wantCount  int
	}{
		{
			name:       "sucesso com resultados",
			input:      dto.ListInput{UserID: userID.String(), Page: 1, Limit: 20},
			repoResult: twoAccounts,
			wantTotal:  2,
			wantCount:  2,
		},
		{
			name:       "sucesso com filtros",
			input:      dto.ListInput{UserID: userID.String(), Page: 1, Limit: 20, Type: "bank_account", ActiveOnly: true},
			repoResult: oneAccount,
			wantTotal:  1,
			wantCount:  1,
		},
		{
			name:       "resultado vazio",
			input:      dto.ListInput{UserID: userID.String(), Page: 1, Limit: 20},
			repoResult: emptyResult,
			wantTotal:  0,
			wantCount:  0,
		},
		{
			name:      "erro do repositório",
			input:     dto.ListInput{UserID: userID.String(), Page: 1, Limit: 20},
			repoErr:   errors.New("database error"),
			wantErr:   true,
			errSubstr: "database error",
		},
		{
			name:      "user_id vazio falha na validação",
			input:     dto.ListInput{UserID: "", Page: 1, Limit: 20},
			wantErr:   true,
			errSubstr: "invalid ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockRepo.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).
				Return(tt.repoResult, tt.repoErr)

			uc := NewListUseCase(mockRepo)
			output, execErr := uc.Execute(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, execErr)
				assert.Nil(t, output)
				if tt.errSubstr != "" {
					assert.Contains(t, execErr.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, execErr)
				assert.NotNil(t, output)
				assert.Len(t, output.Data, tt.wantCount)
				assert.Equal(t, tt.wantTotal, output.Pagination.Total)
			}
		})
	}
}

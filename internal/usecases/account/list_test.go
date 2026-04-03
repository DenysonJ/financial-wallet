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

func TestListUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := uservo.NewID()

	expectedResult := &accountdomain.ListResult{
		Accounts: []*accountdomain.Account{
			{
				ID:        uservo.NewID(),
				UserID:    userID,
				Name:      "Nubank",
				Type:      accountvo.TypeBankAccount,
				Active:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        uservo.NewID(),
				UserID:    userID,
				Name:      "Cartão Inter",
				Type:      accountvo.TypeCreditCard,
				Active:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Total: 2,
		Page:  1,
		Limit: 20,
	}

	mockRepo.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).Return(expectedResult, nil)

	uc := NewListUseCase(mockRepo)
	input := dto.ListInput{UserID: userID.String(), Page: 1, Limit: 20}

	output, listErr := uc.Execute(context.Background(), input)

	assert.NoError(t, listErr)
	assert.NotNil(t, output)
	assert.Len(t, output.Data, 2)
	assert.Equal(t, 2, output.Pagination.Total)
	assert.Equal(t, 1, output.Pagination.Page)
	mockRepo.AssertExpectations(t)
}

func TestListUseCase_Execute_WithFilters(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := uservo.NewID()

	expectedResult := &accountdomain.ListResult{
		Accounts: []*accountdomain.Account{
			{
				ID:        uservo.NewID(),
				UserID:    userID,
				Name:      "Nubank",
				Type:      accountvo.TypeBankAccount,
				Active:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Total: 1,
		Page:  1,
		Limit: 20,
	}

	mockRepo.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).Return(expectedResult, nil)

	uc := NewListUseCase(mockRepo)
	input := dto.ListInput{
		UserID:     userID.String(),
		Page:       1,
		Limit:      20,
		Type:       "bank_account",
		ActiveOnly: true,
	}

	output, listErr := uc.Execute(context.Background(), input)

	assert.NoError(t, listErr)
	assert.Len(t, output.Data, 1)
	assert.Equal(t, "bank_account", output.Data[0].Type)
	mockRepo.AssertExpectations(t)
}

func TestListUseCase_Execute_EmptyResult(t *testing.T) {
	mockRepo := new(MockRepository)

	expectedResult := &accountdomain.ListResult{
		Accounts: []*accountdomain.Account{},
		Total:    0,
		Page:     1,
		Limit:    20,
	}

	mockRepo.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).Return(expectedResult, nil)

	uc := NewListUseCase(mockRepo)
	input := dto.ListInput{UserID: uservo.NewID().String(), Page: 1, Limit: 20}

	output, listErr := uc.Execute(context.Background(), input)

	assert.NoError(t, listErr)
	assert.NotNil(t, output)
	assert.Len(t, output.Data, 0)
	assert.Equal(t, 0, output.Pagination.Total)
	mockRepo.AssertExpectations(t)
}

func TestListUseCase_Execute_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("List", mock.Anything, mock.AnythingOfType("account.ListFilter")).
		Return(nil, errors.New("database error"))

	uc := NewListUseCase(mockRepo)
	input := dto.ListInput{UserID: uservo.NewID().String(), Page: 1, Limit: 20}

	output, listErr := uc.Execute(context.Background(), input)

	assert.Error(t, listErr)
	assert.Nil(t, output)
	assert.Contains(t, listErr.Error(), "database error")
	mockRepo.AssertExpectations(t)
}

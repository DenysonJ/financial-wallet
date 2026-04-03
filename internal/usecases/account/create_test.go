package account

import (
	"context"
	"errors"
	"testing"

	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)

	uc := NewCreateUseCase(mockRepo)
	input := dto.CreateInput{
		UserID:      uservo.NewID().String(),
		Name:        "Nubank",
		Type:        "bank_account",
		Description: "Conta corrente",
	}

	output, createErr := uc.Execute(context.Background(), input)

	assert.NoError(t, createErr)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.ID)
	assert.NotEmpty(t, output.CreatedAt)
	mockRepo.AssertExpectations(t)
}

func TestCreateUseCase_Execute_InvalidUserID(t *testing.T) {
	mockRepo := new(MockRepository)
	uc := NewCreateUseCase(mockRepo)
	input := dto.CreateInput{
		UserID: "invalid-id",
		Name:   "Nubank",
		Type:   "bank_account",
	}

	output, createErr := uc.Execute(context.Background(), input)

	assert.Error(t, createErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, createErr, uservo.ErrInvalidID)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestCreateUseCase_Execute_InvalidType(t *testing.T) {
	mockRepo := new(MockRepository)
	uc := NewCreateUseCase(mockRepo)
	input := dto.CreateInput{
		UserID: uservo.NewID().String(),
		Name:   "Nubank",
		Type:   "savings",
	}

	output, createErr := uc.Execute(context.Background(), input)

	assert.Error(t, createErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, createErr, accountvo.ErrInvalidAccountType)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestCreateUseCase_Execute_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*account.Account")).
		Return(errors.New("database connection failed"))

	uc := NewCreateUseCase(mockRepo)
	input := dto.CreateInput{
		UserID: uservo.NewID().String(),
		Name:   "Nubank",
		Type:   "bank_account",
	}

	output, createErr := uc.Execute(context.Background(), input)

	assert.Error(t, createErr)
	assert.Nil(t, output)
	assert.Contains(t, createErr.Error(), "database connection failed")
	mockRepo.AssertExpectations(t)
}

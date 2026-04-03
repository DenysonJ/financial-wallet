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

func TestGetUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()
	userID := uservo.NewID()

	expected := &accountdomain.Account{
		ID:          id,
		UserID:      userID,
		Name:        "Nubank",
		Type:        accountvo.TypeBankAccount,
		Description: "Conta corrente",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(expected, nil)

	uc := NewGetUseCase(mockRepo)
	output, getErr := uc.Execute(context.Background(), dto.GetInput{ID: id.String()})

	assert.NoError(t, getErr)
	assert.NotNil(t, output)
	assert.Equal(t, id.String(), output.ID)
	assert.Equal(t, userID.String(), output.UserID)
	assert.Equal(t, "Nubank", output.Name)
	assert.Equal(t, "bank_account", output.Type)
	mockRepo.AssertExpectations(t)
}

func TestGetUseCase_Execute_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
		Return(nil, accountdomain.ErrAccountNotFound)

	uc := NewGetUseCase(mockRepo)
	output, getErr := uc.Execute(context.Background(), dto.GetInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456"})

	assert.Error(t, getErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, getErr, accountdomain.ErrAccountNotFound)
	mockRepo.AssertExpectations(t)
}

func TestGetUseCase_Execute_InvalidID(t *testing.T) {
	mockRepo := new(MockRepository)
	uc := NewGetUseCase(mockRepo)

	output, getErr := uc.Execute(context.Background(), dto.GetInput{ID: "invalid-id"})

	assert.Error(t, getErr)
	assert.Nil(t, output)
	mockRepo.AssertNotCalled(t, "FindByID")
}

func TestGetUseCase_Execute_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()
	mockRepo.On("FindByID", mock.Anything, id).Return(nil, errors.New("db error"))

	uc := NewGetUseCase(mockRepo)
	output, getErr := uc.Execute(context.Background(), dto.GetInput{ID: id.String()})

	assert.Error(t, getErr)
	assert.Nil(t, output)
	assert.Contains(t, getErr.Error(), "db error")
	mockRepo.AssertExpectations(t)
}

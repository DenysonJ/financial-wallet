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

func TestDeleteUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()
	mockRepo.On("Delete", mock.Anything, id).Return(nil)

	uc := NewDeleteUseCase(mockRepo)
	output, deleteErr := uc.Execute(context.Background(), dto.DeleteInput{ID: id.String()})

	assert.NoError(t, deleteErr)
	assert.NotNil(t, output)
	assert.Equal(t, id.String(), output.ID)
	assert.NotEmpty(t, output.DeletedAt)
	mockRepo.AssertExpectations(t)
}

func TestDeleteUseCase_Execute_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("vo.ID")).
		Return(accountdomain.ErrAccountNotFound)

	uc := NewDeleteUseCase(mockRepo)
	output, deleteErr := uc.Execute(context.Background(), dto.DeleteInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456"})

	assert.Error(t, deleteErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, deleteErr, accountdomain.ErrAccountNotFound)
	mockRepo.AssertExpectations(t)
}

func TestDeleteUseCase_Execute_InvalidID(t *testing.T) {
	mockRepo := new(MockRepository)
	uc := NewDeleteUseCase(mockRepo)

	output, deleteErr := uc.Execute(context.Background(), dto.DeleteInput{ID: "invalid"})

	assert.Error(t, deleteErr)
	assert.Nil(t, output)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestDeleteUseCase_Execute_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()
	mockRepo.On("Delete", mock.Anything, id).Return(errors.New("db error"))

	uc := NewDeleteUseCase(mockRepo)
	output, deleteErr := uc.Execute(context.Background(), dto.DeleteInput{ID: id.String()})

	assert.Error(t, deleteErr)
	assert.Nil(t, output)
	assert.Contains(t, deleteErr.Error(), "db error")
	mockRepo.AssertExpectations(t)
}

func TestDeleteUseCase_Execute_OwnershipCheck_NotOwner(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()
	ownerID := uservo.NewID()
	otherUserID := uservo.NewID()

	existing := &accountdomain.Account{
		ID:        id,
		UserID:    ownerID,
		Name:      "Nubank",
		Type:      accountvo.TypeBankAccount,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existing, nil)

	uc := NewDeleteUseCase(mockRepo)
	output, deleteErr := uc.Execute(context.Background(), dto.DeleteInput{
		ID:               id.String(),
		RequestingUserID: otherUserID.String(),
	})

	assert.Error(t, deleteErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, deleteErr, accountdomain.ErrAccountNotFound)
	mockRepo.AssertNotCalled(t, "Delete")
}

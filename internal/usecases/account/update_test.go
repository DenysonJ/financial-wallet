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

func TestUpdateUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()
	userID := uservo.NewID()

	existing := &accountdomain.Account{
		ID:          id,
		UserID:      userID,
		Name:        "Nubank",
		Type:        accountvo.TypeBankAccount,
		Description: "Old description",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existing, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)

	uc := NewUpdateUseCase(mockRepo)
	newName := "Nubank Ultravioleta"
	newDesc := "Conta premium"
	input := dto.UpdateInput{
		ID:          id.String(),
		Name:        &newName,
		Description: &newDesc,
	}

	output, updateErr := uc.Execute(context.Background(), input)

	assert.NoError(t, updateErr)
	assert.NotNil(t, output)
	assert.Equal(t, "Nubank Ultravioleta", output.Name)
	assert.Equal(t, "Conta premium", output.Description)
	assert.Equal(t, "bank_account", output.Type)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUseCase_Execute_PartialUpdate_NameOnly(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()

	existing := &accountdomain.Account{
		ID:          id,
		UserID:      uservo.NewID(),
		Name:        "Old Name",
		Type:        accountvo.TypeCash,
		Description: "Keep this",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existing, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).Return(nil)

	uc := NewUpdateUseCase(mockRepo)
	newName := "New Name"
	input := dto.UpdateInput{ID: id.String(), Name: &newName}

	output, updateErr := uc.Execute(context.Background(), input)

	assert.NoError(t, updateErr)
	assert.Equal(t, "New Name", output.Name)
	assert.Equal(t, "Keep this", output.Description)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUseCase_Execute_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
		Return(nil, accountdomain.ErrAccountNotFound)

	uc := NewUpdateUseCase(mockRepo)
	newName := "New Name"
	input := dto.UpdateInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456", Name: &newName}

	output, updateErr := uc.Execute(context.Background(), input)

	assert.Error(t, updateErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, updateErr, accountdomain.ErrAccountNotFound)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUpdateUseCase_Execute_InvalidID(t *testing.T) {
	mockRepo := new(MockRepository)
	uc := NewUpdateUseCase(mockRepo)
	newName := "New Name"
	input := dto.UpdateInput{ID: "invalid", Name: &newName}

	output, updateErr := uc.Execute(context.Background(), input)

	assert.Error(t, updateErr)
	assert.Nil(t, output)
	mockRepo.AssertNotCalled(t, "FindByID")
}

func TestUpdateUseCase_Execute_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	id := uservo.NewID()

	existing := &accountdomain.Account{
		ID:        id,
		UserID:    uservo.NewID(),
		Name:      "Nubank",
		Type:      accountvo.TypeBankAccount,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existing, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*account.Account")).
		Return(errors.New("db error"))

	uc := NewUpdateUseCase(mockRepo)
	newName := "Updated"
	input := dto.UpdateInput{ID: id.String(), Name: &newName}

	output, updateErr := uc.Execute(context.Background(), input)

	assert.Error(t, updateErr)
	assert.Nil(t, output)
	assert.Contains(t, updateErr.Error(), "db error")
	mockRepo.AssertExpectations(t)
}

func TestUpdateUseCase_Execute_OwnershipCheck_NotOwner(t *testing.T) {
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

	uc := NewUpdateUseCase(mockRepo)
	newName := "Hacked"
	input := dto.UpdateInput{
		ID:               id.String(),
		RequestingUserID: otherUserID.String(),
		Name:             &newName,
	}

	output, updateErr := uc.Execute(context.Background(), input)

	assert.Error(t, updateErr)
	assert.Nil(t, output)
	assert.ErrorIs(t, updateErr, accountdomain.ErrAccountNotFound)
	mockRepo.AssertNotCalled(t, "Update")
}

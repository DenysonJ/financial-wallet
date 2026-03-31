package user

import (
	"context"
	"testing"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSetPasswordUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := vo.NewID()
	existingUser := &userdomain.User{
		ID:           userID,
		Name:         "Test User",
		Email:        vo.ParseEmail("test@example.com"),
		PasswordHash: "", // no password set
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, userID).Return(existingUser, nil)
	mockRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)

	uc := NewSetPasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.SetPasswordInput{
		UserID:               userID.String(),
		Password:             "Str0ng!Pass",
		PasswordConfirmation: "Str0ng!Pass",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.NoError(t, execErr)
	mockRepo.AssertExpectations(t)
}

func TestSetPasswordUseCase_Execute_PasswordAlreadySet(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := vo.NewID()
	existingUser := &userdomain.User{
		ID:           userID,
		Name:         "Test User",
		Email:        vo.ParseEmail("test@example.com"),
		PasswordHash: "$2a$12$existinghash",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, userID).Return(existingUser, nil)

	uc := NewSetPasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.SetPasswordInput{
		UserID:               userID.String(),
		Password:             "Str0ng!Pass",
		PasswordConfirmation: "Str0ng!Pass",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, execErr, userdomain.ErrPasswordAlreadySet)
	mockRepo.AssertNotCalled(t, "UpdatePassword")
}

func TestSetPasswordUseCase_Execute_PasswordMismatch(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := vo.NewID()
	existingUser := &userdomain.User{
		ID:           userID,
		Name:         "Test User",
		Email:        vo.ParseEmail("test@example.com"),
		PasswordHash: "",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, userID).Return(existingUser, nil)

	uc := NewSetPasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.SetPasswordInput{
		UserID:               userID.String(),
		Password:             "Str0ng!Pass",
		PasswordConfirmation: "Different1!",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, execErr, userdomain.ErrPasswordMismatch)
	mockRepo.AssertNotCalled(t, "UpdatePassword")
}

func TestSetPasswordUseCase_Execute_PasswordTooShort(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := vo.NewID()
	existingUser := &userdomain.User{
		ID:           userID,
		Name:         "Test User",
		Email:        vo.ParseEmail("test@example.com"),
		PasswordHash: "",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, userID).Return(existingUser, nil)

	uc := NewSetPasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.SetPasswordInput{
		UserID:               userID.String(),
		Password:             "Ab1!",
		PasswordConfirmation: "Ab1!",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, execErr, vo.ErrPasswordTooShort)
	mockRepo.AssertNotCalled(t, "UpdatePassword")
}

func TestSetPasswordUseCase_Execute_UserNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	userID := vo.NewID()

	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, userdomain.ErrUserNotFound)

	uc := NewSetPasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.SetPasswordInput{
		UserID:               userID.String(),
		Password:             "Str0ng!Pass",
		PasswordConfirmation: "Str0ng!Pass",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, execErr, userdomain.ErrUserNotFound)
}

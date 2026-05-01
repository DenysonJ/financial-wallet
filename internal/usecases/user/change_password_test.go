package user

import (
	"context"
	"testing"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/useruci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newUserWithPassword(t *testing.T) (*userdomain.User, string) {
	t.Helper()
	plain := "OldPassword1!"
	pw, hashErr := vo.NewPassword(plain, 4)
	assert.NoError(t, hashErr)
	return &userdomain.User{
		ID:           vo.NewID(),
		Name:         "Test User",
		Email:        vo.ParseEmail("test@example.com"),
		PasswordHash: pw.String(),
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, plain
}

func TestChangePasswordUseCase_Execute_Success(t *testing.T) {
	mockRepo := useruci.NewMockRepository(t)
	existingUser, oldPlain := newUserWithPassword(t)

	mockRepo.On("FindByID", mock.Anything, existingUser.ID).Return(existingUser, nil)
	mockRepo.On("UpdatePassword", mock.Anything, existingUser.ID, mock.AnythingOfType("string")).Return(nil)

	uc := NewChangePasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.ChangePasswordInput{
		UserID:                  existingUser.ID.String(),
		CurrentPassword:         oldPlain,
		NewPassword:             "NewStr0ng!Pass",
		NewPasswordConfirmation: "NewStr0ng!Pass",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.NoError(t, execErr)
	mockRepo.AssertExpectations(t)
}

func TestChangePasswordUseCase_Execute_WrongCurrentPassword(t *testing.T) {
	mockRepo := useruci.NewMockRepository(t)
	existingUser, _ := newUserWithPassword(t)

	mockRepo.On("FindByID", mock.Anything, existingUser.ID).Return(existingUser, nil)

	uc := NewChangePasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.ChangePasswordInput{
		UserID:                  existingUser.ID.String(),
		CurrentPassword:         "WrongPassword1!",
		NewPassword:             "NewStr0ng!Pass",
		NewPasswordConfirmation: "NewStr0ng!Pass",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	mockRepo.AssertNotCalled(t, "UpdatePassword")
}

func TestChangePasswordUseCase_Execute_NewPasswordMismatch(t *testing.T) {
	mockRepo := useruci.NewMockRepository(t)
	existingUser, oldPlain := newUserWithPassword(t)

	mockRepo.On("FindByID", mock.Anything, existingUser.ID).Return(existingUser, nil)

	uc := NewChangePasswordUseCase(mockRepo).WithBcryptCost(4)
	input := dto.ChangePasswordInput{
		UserID:                  existingUser.ID.String(),
		CurrentPassword:         oldPlain,
		NewPassword:             "NewStr0ng!Pass",
		NewPasswordConfirmation: "Different1!Diff",
	}

	execErr := uc.Execute(context.Background(), input)

	assert.ErrorIs(t, execErr, userdomain.ErrPasswordMismatch)
	mockRepo.AssertNotCalled(t, "UpdatePassword")
}

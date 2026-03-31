package auth

import (
	"context"
	"testing"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeUserWithPassword(t *testing.T) *userdomain.User {
	t.Helper()
	pw, hashErr := vo.NewPassword("Str0ng!Pass", 4)
	assert.NoError(t, hashErr)
	return &userdomain.User{
		ID:           vo.NewID(),
		Name:         "Test User",
		Email:        vo.ParseEmail("test@example.com"),
		PasswordHash: pw.String(),
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestLoginUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	user := makeUserWithPassword(t)

	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
	mockToken.On("GenerateAccessToken", user.ID.String()).Return("access-token", nil)
	mockToken.On("GenerateRefreshToken", user.ID.String()).Return("refresh-token", nil)

	uc := NewLoginUseCase(mockRepo, mockToken)
	output, execErr := uc.Execute(context.Background(), dto.LoginInput{
		Email:    "test@example.com",
		Password: "Str0ng!Pass",
	})

	assert.NoError(t, execErr)
	assert.Equal(t, "access-token", output.AccessToken)
	assert.Equal(t, "refresh-token", output.RefreshToken)
	mockRepo.AssertExpectations(t)
	mockToken.AssertExpectations(t)
}

func TestLoginUseCase_Execute_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)

	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, userdomain.ErrUserNotFound)

	uc := NewLoginUseCase(mockRepo, mockToken)
	output, execErr := uc.Execute(context.Background(), dto.LoginInput{
		Email:    "notfound@example.com",
		Password: "Str0ng!Pass",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
}

func TestLoginUseCase_Execute_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	user := makeUserWithPassword(t)

	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)

	uc := NewLoginUseCase(mockRepo, mockToken)
	output, execErr := uc.Execute(context.Background(), dto.LoginInput{
		Email:    "test@example.com",
		Password: "WrongPass1!",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
	mockToken.AssertNotCalled(t, "GenerateAccessToken")
}

func TestLoginUseCase_Execute_InactiveUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	user := makeUserWithPassword(t)
	user.Active = false

	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)

	uc := NewLoginUseCase(mockRepo, mockToken)
	output, execErr := uc.Execute(context.Background(), dto.LoginInput{
		Email:    "test@example.com",
		Password: "Str0ng!Pass",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
}

func TestLoginUseCase_Execute_NoPasswordSet(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	user := makeUserWithPassword(t)
	user.PasswordHash = "" // no password

	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)

	uc := NewLoginUseCase(mockRepo, mockToken)
	output, execErr := uc.Execute(context.Background(), dto.LoginInput{
		Email:    "test@example.com",
		Password: "Str0ng!Pass",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
}

func TestLoginUseCase_Execute_InvalidEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)

	uc := NewLoginUseCase(mockRepo, mockToken)
	output, execErr := uc.Execute(context.Background(), dto.LoginInput{
		Email:    "not-an-email",
		Password: "Str0ng!Pass",
	})

	assert.ErrorIs(t, execErr, userdomain.ErrInvalidCredentials)
	assert.Nil(t, output)
}

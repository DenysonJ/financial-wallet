package auth

import (
	"context"
	"testing"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/authuci"
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

func TestLoginUseCase_Execute(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(repo *authuci.MockUserRepository, token *authuci.MockTokenService, user *userdomain.User)
		mutateUser func(user *userdomain.User)
		email      string
		password   string
		wantErr    error
		wantTokens bool
	}{
		{
			name: "success",
			setupMock: func(repo *authuci.MockUserRepository, token *authuci.MockTokenService, user *userdomain.User) {
				repo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
				token.On("GenerateAccessToken", user.ID.String()).Return("access-token", nil)
				token.On("GenerateRefreshToken", user.ID.String()).Return("refresh-token", nil)
			},
			email:      "test@example.com",
			password:   "Str0ng!Pass",
			wantTokens: true,
		},
		{
			name: "user not found",
			setupMock: func(repo *authuci.MockUserRepository, token *authuci.MockTokenService, _ *userdomain.User) {
				repo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, userdomain.ErrUserNotFound)
			},
			email:    "notfound@example.com",
			password: "Str0ng!Pass",
			wantErr:  userdomain.ErrInvalidCredentials,
		},
		{
			name: "wrong password",
			setupMock: func(repo *authuci.MockUserRepository, _ *authuci.MockTokenService, user *userdomain.User) {
				repo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
			},
			email:    "test@example.com",
			password: "WrongPass1!",
			wantErr:  userdomain.ErrInvalidCredentials,
		},
		{
			name: "inactive user",
			setupMock: func(repo *authuci.MockUserRepository, _ *authuci.MockTokenService, user *userdomain.User) {
				repo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
			},
			mutateUser: func(user *userdomain.User) { user.Active = false },
			email:      "test@example.com",
			password:   "Str0ng!Pass",
			wantErr:    userdomain.ErrInvalidCredentials,
		},
		{
			name: "no password set",
			setupMock: func(repo *authuci.MockUserRepository, _ *authuci.MockTokenService, user *userdomain.User) {
				repo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil)
			},
			mutateUser: func(user *userdomain.User) { user.PasswordHash = "" },
			email:      "test@example.com",
			password:   "Str0ng!Pass",
			wantErr:    userdomain.ErrInvalidCredentials,
		},
		{
			name: "invalid email",
			setupMock: func(_ *authuci.MockUserRepository, _ *authuci.MockTokenService, _ *userdomain.User) {
				// no mock setup needed — validation fails before repo call
			},
			email:    "not-an-email",
			password: "Str0ng!Pass",
			wantErr:  userdomain.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := authuci.NewMockUserRepository(t)
			mockToken := authuci.NewMockTokenService(t)
			user := makeUserWithPassword(t)

			if tt.mutateUser != nil {
				tt.mutateUser(user)
			}
			tt.setupMock(mockRepo, mockToken, user)

			uc := NewLoginUseCase(mockRepo, mockToken)
			output, execErr := uc.Execute(context.Background(), dto.LoginInput{
				Email:    tt.email,
				Password: tt.password,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, execErr)
			}

			if tt.wantTokens {
				assert.Equal(t, "access-token", output.AccessToken)
				assert.Equal(t, "refresh-token", output.RefreshToken)
			}

			mockRepo.AssertExpectations(t)
			mockToken.AssertExpectations(t)
		})
	}
}

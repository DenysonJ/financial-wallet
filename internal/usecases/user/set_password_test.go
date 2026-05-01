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

func TestSetPasswordUseCase_Execute(t *testing.T) {
	makeUser := func(id vo.ID, passwordHash string) *userdomain.User {
		return &userdomain.User{
			ID:           id,
			Name:         "Test User",
			Email:        vo.ParseEmail("test@example.com"),
			PasswordHash: passwordHash,
			Active:       true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	tests := []struct {
		name       string
		setupMock  func(repo *useruci.MockRepository, userID vo.ID)
		password   string
		confirm    string
		wantErr    error
		wantUpdate bool
	}{
		{
			name: "success",
			setupMock: func(repo *useruci.MockRepository, userID vo.ID) {
				repo.On("FindByID", mock.Anything, userID).Return(makeUser(userID, ""), nil)
				repo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)
			},
			password:   "Str0ng!Passw",
			confirm:    "Str0ng!Passw",
			wantErr:    nil,
			wantUpdate: true,
		},
		{
			name: "password already set",
			setupMock: func(repo *useruci.MockRepository, userID vo.ID) {
				repo.On("FindByID", mock.Anything, userID).Return(makeUser(userID, "$2a$12$existinghash"), nil)
			},
			password: "Str0ng!Passw",
			confirm:  "Str0ng!Passw",
			wantErr:  userdomain.ErrPasswordAlreadySet,
		},
		{
			name: "password mismatch",
			setupMock: func(repo *useruci.MockRepository, userID vo.ID) {
				repo.On("FindByID", mock.Anything, userID).Return(makeUser(userID, ""), nil)
			},
			password: "Str0ng!Passw",
			confirm:  "DifferentPa1!",
			wantErr:  userdomain.ErrPasswordMismatch,
		},
		{
			name: "password too short",
			setupMock: func(repo *useruci.MockRepository, userID vo.ID) {
				repo.On("FindByID", mock.Anything, userID).Return(makeUser(userID, ""), nil)
			},
			password: "Ab1!",
			confirm:  "Ab1!",
			wantErr:  vo.ErrPasswordTooShort,
		},
		{
			name: "user not found",
			setupMock: func(repo *useruci.MockRepository, userID vo.ID) {
				repo.On("FindByID", mock.Anything, userID).Return(nil, userdomain.ErrUserNotFound)
			},
			password: "Str0ng!Passw",
			confirm:  "Str0ng!Passw",
			wantErr:  userdomain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := useruci.NewMockRepository(t)
			userID := vo.NewID()
			tt.setupMock(mockRepo, userID)

			uc := NewSetPasswordUseCase(mockRepo).WithBcryptCost(4)
			input := dto.SetPasswordInput{
				UserID:               userID.String(),
				Password:             tt.password,
				PasswordConfirmation: tt.confirm,
			}

			execErr := uc.Execute(context.Background(), input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
			} else {
				assert.NoError(t, execErr)
			}

			if !tt.wantUpdate {
				mockRepo.AssertNotCalled(t, "UpdatePassword")
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

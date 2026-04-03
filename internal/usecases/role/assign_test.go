package role

import (
	"context"
	"testing"
	"time"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAssignRoleUseCase_Execute(t *testing.T) {
	validUserID := vo.NewID()
	validRoleID := vo.NewID()
	existingRole := &roledomain.Role{
		ID:        validRoleID,
		Name:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name      string
		userID    string
		roleID    string
		setupMock func(repo *MockRepository)
		wantErr   error
	}{
		{
			name:   "success",
			userID: validUserID.String(),
			roleID: validRoleID.String(),
			setupMock: func(repo *MockRepository) {
				repo.On("FindByID", mock.Anything, validRoleID).Return(existingRole, nil)
				repo.On("AssignRole", mock.Anything, validUserID, validRoleID).Return(nil)
			},
		},
		{
			name:      "invalid user ID",
			userID:    "not-a-uuid",
			roleID:    validRoleID.String(),
			setupMock: func(_ *MockRepository) {},
			wantErr:   vo.ErrInvalidID,
		},
		{
			name:      "invalid role ID",
			userID:    validUserID.String(),
			roleID:    "not-a-uuid",
			setupMock: func(_ *MockRepository) {},
			wantErr:   vo.ErrInvalidID,
		},
		{
			name:   "role not found",
			userID: validUserID.String(),
			roleID: validRoleID.String(),
			setupMock: func(repo *MockRepository) {
				repo.On("FindByID", mock.Anything, validRoleID).Return(nil, roledomain.ErrRoleNotFound)
			},
			wantErr: roledomain.ErrRoleNotFound,
		},
		{
			name:   "role already assigned",
			userID: validUserID.String(),
			roleID: validRoleID.String(),
			setupMock: func(repo *MockRepository) {
				repo.On("FindByID", mock.Anything, validRoleID).Return(existingRole, nil)
				repo.On("AssignRole", mock.Anything, validUserID, validRoleID).Return(roledomain.ErrRoleAlreadyAssigned)
			},
			wantErr: roledomain.ErrRoleAlreadyAssigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)

			uc := NewAssignRoleUseCase(mockRepo)
			execErr := uc.Execute(context.Background(), dto.AssignRoleInput{
				UserID: tt.userID,
				RoleID: tt.roleID,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
			} else {
				assert.NoError(t, execErr)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

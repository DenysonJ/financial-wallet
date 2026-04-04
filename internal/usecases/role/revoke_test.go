package role

import (
	"context"
	"testing"
	"time"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/roleuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRevokeRoleUseCase_Execute(t *testing.T) {
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
		setupMock func(repo *roleuci.MockRepository)
		wantErr   error
	}{
		{
			name:   "success",
			userID: validUserID.String(),
			roleID: validRoleID.String(),
			setupMock: func(repo *roleuci.MockRepository) {
				repo.On("FindByID", mock.Anything, validRoleID).Return(existingRole, nil)
				repo.On("RevokeRole", mock.Anything, validUserID, validRoleID).Return(nil)
			},
		},
		{
			name:      "invalid user ID",
			userID:    "not-a-uuid",
			roleID:    validRoleID.String(),
			setupMock: func(_ *roleuci.MockRepository) {},
			wantErr:   vo.ErrInvalidID,
		},
		{
			name:      "invalid role ID",
			userID:    validUserID.String(),
			roleID:    "not-a-uuid",
			setupMock: func(_ *roleuci.MockRepository) {},
			wantErr:   vo.ErrInvalidID,
		},
		{
			name:   "role not found",
			userID: validUserID.String(),
			roleID: validRoleID.String(),
			setupMock: func(repo *roleuci.MockRepository) {
				repo.On("FindByID", mock.Anything, validRoleID).Return(nil, roledomain.ErrRoleNotFound)
			},
			wantErr: roledomain.ErrRoleNotFound,
		},
		{
			name:   "role not assigned",
			userID: validUserID.String(),
			roleID: validRoleID.String(),
			setupMock: func(repo *roleuci.MockRepository) {
				repo.On("FindByID", mock.Anything, validRoleID).Return(existingRole, nil)
				repo.On("RevokeRole", mock.Anything, validUserID, validRoleID).Return(roledomain.ErrRoleNotAssigned)
			},
			wantErr: roledomain.ErrRoleNotAssigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := roleuci.NewMockRepository(t)
			tt.setupMock(mockRepo)

			uc := NewRevokeRoleUseCase(mockRepo)
			execErr := uc.Execute(context.Background(), dto.RevokeRoleInput{
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

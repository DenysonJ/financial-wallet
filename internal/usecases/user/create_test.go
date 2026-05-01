package user

import (
	"context"
	"errors"
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/useruci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateUseCase_Execute(t *testing.T) {
	tests := []struct {
		name            string
		input           dto.CreateInput
		setupMock       func(repo *useruci.MockRepository)
		wantErr         error
		wantErrContains string
		wantOutput      bool
	}{
		{
			name:  "GIVEN valid input WHEN executing THEN succeeds",
			input: dto.CreateInput{Name: "João Silva", Email: "joao@example.com"},
			setupMock: func(repo *useruci.MockRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantOutput: true,
		},
		{
			name:  "GIVEN invalid email WHEN executing THEN returns ErrInvalidEmail",
			input: dto.CreateInput{Name: "João Silva", Email: "invalid-email"},
			setupMock: func(_ *useruci.MockRepository) {
				// no repo call expected
			},
			wantErr: vo.ErrInvalidEmail,
		},
		{
			name:  "GIVEN repository failure WHEN executing THEN propagates error",
			input: dto.CreateInput{Name: "João Silva", Email: "joao@example.com"},
			setupMock: func(repo *useruci.MockRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).
					Return(errors.New("database connection failed"))
			},
			wantErrContains: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := useruci.NewMockRepository(t)
			tt.setupMock(mockRepo)

			uc := NewCreateUseCase(mockRepo)
			output, execErr := uc.Execute(context.Background(), tt.input)

			switch {
			case tt.wantErr != nil:
				assert.ErrorIs(t, execErr, tt.wantErr)
				assert.Nil(t, output)
			case tt.wantErrContains != "":
				assert.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.wantErrContains)
				assert.Nil(t, output)
			default:
				assert.NoError(t, execErr)
			}

			if tt.wantOutput {
				assert.NotNil(t, output)
				assert.NotEmpty(t, output.ID)
				assert.NotEmpty(t, output.CreatedAt)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

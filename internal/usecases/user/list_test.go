package user

import (
	"context"
	"errors"
	"testing"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/useruci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListUseCase_Execute(t *testing.T) {
	now := time.Now()
	email1, _ := vo.NewEmail("joao@example.com")
	email2, _ := vo.NewEmail("maria@example.com")

	twoUsers := &userdomain.ListResult{
		Users: []*userdomain.User{
			{ID: vo.NewID(), Name: "João Silva", Email: email1, Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: vo.NewID(), Name: "Maria Santos", Email: email2, Active: true, CreatedAt: now, UpdatedAt: now},
		},
		Total: 2, Page: 1, Limit: 20,
	}
	oneUser := &userdomain.ListResult{
		Users: []*userdomain.User{
			{ID: vo.NewID(), Name: "Maria Santos", Email: email2, Active: true, CreatedAt: now, UpdatedAt: now},
		},
		Total: 1, Page: 1, Limit: 20,
	}
	empty := &userdomain.ListResult{Users: []*userdomain.User{}, Total: 0, Page: 1, Limit: 20}

	tests := []struct {
		name            string
		input           dto.ListInput
		repoResult      *userdomain.ListResult
		repoErr         error
		wantLen         int
		wantTotal       int
		wantFirstName   string
		wantErrContains string
	}{
		{
			name:       "GIVEN no filters WHEN executing THEN returns all users",
			input:      dto.ListInput{Page: 1, Limit: 20},
			repoResult: twoUsers,
			wantLen:    2, wantTotal: 2,
		},
		{
			name:          "GIVEN name+active filters WHEN executing THEN returns matched users",
			input:         dto.ListInput{Page: 1, Limit: 20, Name: "maria", ActiveOnly: true},
			repoResult:    oneUser,
			wantLen:       1,
			wantTotal:     1,
			wantFirstName: "Maria Santos",
		},
		{
			name:       "GIVEN repository empty WHEN executing THEN returns empty page",
			input:      dto.ListInput{Page: 1, Limit: 20},
			repoResult: empty,
			wantLen:    0, wantTotal: 0,
		},
		{
			name:            "GIVEN repository failure WHEN executing THEN propagates error",
			input:           dto.ListInput{Page: 1, Limit: 20},
			repoErr:         errors.New("database error"),
			wantErrContains: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := useruci.NewMockRepository(t)
			mockRepo.On("List", mock.Anything, mock.AnythingOfType("user.ListFilter")).
				Return(tt.repoResult, tt.repoErr)

			uc := NewListUseCase(mockRepo)
			output, execErr := uc.Execute(context.Background(), tt.input)

			if tt.wantErrContains != "" {
				assert.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.wantErrContains)
				assert.Nil(t, output)
				mockRepo.AssertExpectations(t)
				return
			}

			assert.NoError(t, execErr)
			assert.NotNil(t, output)
			assert.Len(t, output.Data, tt.wantLen)
			assert.Equal(t, tt.wantTotal, output.Pagination.Total)
			if tt.wantFirstName != "" {
				assert.Equal(t, tt.wantFirstName, output.Data[0].Name)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

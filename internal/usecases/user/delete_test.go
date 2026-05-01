package user

import (
	"context"
	"errors"
	"testing"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/useruci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteUseCase_Execute(t *testing.T) {
	validID := vo.NewID()
	cacheKey := "user:" + validID.String()

	tests := []struct {
		name            string
		input           dto.DeleteInput
		withCache       bool
		setupMock       func(repo *useruci.MockRepository, cache *useruci.MockCache)
		wantErr         error
		wantErrContains string
		wantOutput      bool
	}{
		{
			name:  "GIVEN valid ID WHEN executing THEN soft-deletes and returns output",
			input: dto.DeleteInput{ID: validID.String()},
			setupMock: func(repo *useruci.MockRepository, _ *useruci.MockCache) {
				repo.On("Delete", mock.Anything, validID).Return(nil)
			},
			wantOutput: true,
		},
		{
			name:  "GIVEN repository returns ErrUserNotFound WHEN executing THEN propagates",
			input: dto.DeleteInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456"},
			setupMock: func(repo *useruci.MockRepository, _ *useruci.MockCache) {
				repo.On("Delete", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(userdomain.ErrUserNotFound)
			},
			wantErr: userdomain.ErrUserNotFound,
		},
		{
			name:  "GIVEN invalid ID WHEN executing THEN does not call repo",
			input: dto.DeleteInput{ID: "invalid-id"},
			setupMock: func(_ *useruci.MockRepository, _ *useruci.MockCache) {
				// no repo call expected
			},
			wantErr: vo.ErrInvalidID,
		},
		{
			name:      "GIVEN cache delete fails WHEN executing THEN delete still succeeds",
			input:     dto.DeleteInput{ID: validID.String()},
			withCache: true,
			setupMock: func(repo *useruci.MockRepository, cache *useruci.MockCache) {
				repo.On("Delete", mock.Anything, validID).Return(nil)
				cache.On("Delete", mock.Anything, cacheKey).Return(errors.New("redis connection refused"))
			},
			wantOutput: true,
		},
		{
			name:      "GIVEN cache available WHEN executing THEN invalidates cache key",
			input:     dto.DeleteInput{ID: validID.String()},
			withCache: true,
			setupMock: func(repo *useruci.MockRepository, cache *useruci.MockCache) {
				repo.On("Delete", mock.Anything, validID).Return(nil)
				cache.On("Delete", mock.Anything, cacheKey).Return(nil)
			},
			wantOutput: true,
		},
		{
			name:  "GIVEN repository returns generic error WHEN executing THEN propagates",
			input: dto.DeleteInput{ID: "018e4a2c-6b4d-7000-9410-abcdef123456"},
			setupMock: func(repo *useruci.MockRepository, _ *useruci.MockCache) {
				repo.On("Delete", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(errors.New("database error"))
			},
			wantErrContains: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := useruci.NewMockRepository(t)
			var mockCache *useruci.MockCache
			if tt.withCache {
				mockCache = useruci.NewMockCache(t)
			}
			tt.setupMock(mockRepo, mockCache)

			uc := NewDeleteUseCase(mockRepo)
			if tt.withCache {
				uc = uc.WithCache(mockCache)
			}

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
				assert.NotEmpty(t, output.DeletedAt)
			}

			mockRepo.AssertExpectations(t)
			if mockCache != nil {
				mockCache.AssertExpectations(t)
			}
		})
	}
}

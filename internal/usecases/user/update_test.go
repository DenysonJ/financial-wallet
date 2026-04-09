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

func TestUpdateUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := useruci.NewMockRepository(t)
	id := vo.NewID()
	email, _ := vo.NewEmail("joao@example.com")

	existingEntity := &userdomain.User{
		ID:        id,
		Name:      "João Silva",
		Email:     email,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existingEntity, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)

	uc := NewUpdateUseCase(mockRepo)
	input := dto.UpdateInput{
		ID:   id.String(),
		Name: new("João Silva Updated"),
	}

	// Act
	output, err := uc.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "João Silva Updated", output.Name)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUseCase_Execute_NotFound(t *testing.T) {
	// Arrange
	mockRepo := useruci.NewMockRepository(t)
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
		Return(nil, userdomain.ErrUserNotFound)

	uc := NewUpdateUseCase(mockRepo)
	input := dto.UpdateInput{
		ID:   "018e4a2c-6b4d-7000-9410-abcdef123456",
		Name: new("Updated Name"),
	}

	// Act
	output, err := uc.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, userdomain.ErrUserNotFound))
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUpdateUseCase_Execute_InvalidEmail(t *testing.T) {
	// Arrange
	mockRepo := useruci.NewMockRepository(t)
	id := vo.NewID()
	email, _ := vo.NewEmail("joao@example.com")

	existingEntity := &userdomain.User{
		ID:        id,
		Name:      "João Silva",
		Email:     email,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existingEntity, nil)

	uc := NewUpdateUseCase(mockRepo)
	input := dto.UpdateInput{
		ID:    id.String(),
		Email: new("invalid-email"),
	}

	// Act
	output, err := uc.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, vo.ErrInvalidEmail))
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUpdateUseCase_Execute_InvalidID(t *testing.T) {
	// Arrange
	mockRepo := useruci.NewMockRepository(t)
	uc := NewUpdateUseCase(mockRepo)
	input := dto.UpdateInput{
		ID:   "invalid-id",
		Name: new("Updated Name"),
	}

	// Act
	output, err := uc.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	mockRepo.AssertNotCalled(t, "FindByID")
}

func TestUpdateUseCase_Execute_CacheDeleteError_StillSucceeds(t *testing.T) {
	// Arrange
	mockRepo := useruci.NewMockRepository(t)
	mockCache := useruci.NewMockCache(t)
	id := vo.NewID()
	email, _ := vo.NewEmail("joao@example.com")
	cacheKey := "user:" + id.String()

	existingEntity := &userdomain.User{
		ID:        id,
		Name:      "João Silva",
		Email:     email,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existingEntity, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
	mockCache.On("Delete", mock.Anything, cacheKey).Return(errors.New("redis connection refused"))

	uc := NewUpdateUseCase(mockRepo).WithCache(mockCache)
	input := dto.UpdateInput{
		ID:   id.String(),
		Name: new("João Silva Updated"),
	}

	// Act
	output, updateErr := uc.Execute(context.Background(), input)

	// Assert — update succeeds even though cache delete failed
	assert.NoError(t, updateErr)
	assert.NotNil(t, output)
	assert.Equal(t, "João Silva Updated", output.Name)
	mockCache.AssertCalled(t, "Delete", mock.Anything, cacheKey)
}

func TestUpdateUseCase_Execute_CacheInvalidation(t *testing.T) {
	// Arrange
	mockRepo := useruci.NewMockRepository(t)
	mockCache := useruci.NewMockCache(t)
	id := vo.NewID()
	email, _ := vo.NewEmail("joao@example.com")
	cacheKey := "user:" + id.String()

	existingEntity := &userdomain.User{
		ID:        id,
		Name:      "João Silva",
		Email:     email,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("FindByID", mock.Anything, id).Return(existingEntity, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
	mockCache.On("Delete", mock.Anything, cacheKey).Return(nil)

	uc := NewUpdateUseCase(mockRepo).WithCache(mockCache)
	input := dto.UpdateInput{
		ID:   id.String(),
		Name: new("João Silva Updated"),
	}

	// Act
	output, err := uc.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	mockCache.AssertCalled(t, "Delete", mock.Anything, cacheKey)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

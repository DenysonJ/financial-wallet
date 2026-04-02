package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
)

// DeleteUseCase implementa o caso de uso de deleção (soft delete) de user.
type DeleteUseCase struct {
	repo  interfaces.Repository
	cache interfaces.Cache
}

// NewDeleteUseCase cria uma nova instância do DeleteUseCase.
func NewDeleteUseCase(repo interfaces.Repository) *DeleteUseCase {
	return &DeleteUseCase{
		repo: repo,
	}
}

// WithCache sets an optional cache for the use case (builder pattern).
func (uc *DeleteUseCase) WithCache(c interfaces.Cache) *DeleteUseCase {
	uc.cache = c
	return uc
}

// Execute realiza soft delete de um user.
//
// Fluxo:
//  1. Validar ID
//  2. Realizar soft delete (active=false)
//  3. Invalidar cache
func (uc *DeleteUseCase) Execute(ctx context.Context, input dto.DeleteInput) (*dto.DeleteOutput, error) {
	// Validar e converter ID
	id, err := vo.ParseID(input.ID)
	if err != nil {
		return nil, err
	}

	// 2. Realizar soft delete
	if err := uc.repo.Delete(ctx, id); err != nil {
		return nil, err
	}

	// 3. Invalidar cache
	if uc.cache != nil {
		cacheKey := "user:" + input.ID
		if err := uc.cache.Delete(ctx, cacheKey); err != nil {
			slog.Warn("failed to invalidate cache", "key", cacheKey, "error", err)
		}
	}

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

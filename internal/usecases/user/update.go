package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
)

// UpdateUseCase implementa o caso de uso de atualização de user.
type UpdateUseCase struct {
	repo  interfaces.Repository
	cache interfaces.Cache
}

// NewUpdateUseCase cria uma nova instância do UpdateUseCase.
func NewUpdateUseCase(repo interfaces.Repository) *UpdateUseCase {
	return &UpdateUseCase{
		repo: repo,
	}
}

// WithCache sets an optional cache for the use case (builder pattern).
func (uc *UpdateUseCase) WithCache(c interfaces.Cache) *UpdateUseCase {
	uc.cache = c
	return uc
}

// Execute atualiza um user existente.
//
// Fluxo:
//  1. Buscar user existente pelo ID
//  2. Aplicar atualizações parciais
//  3. Persistir alterações
//  4. Invalidar cache
func (uc *UpdateUseCase) Execute(ctx context.Context, input dto.UpdateInput) (*dto.UpdateOutput, error) {
	// Validar e converter ID
	id, err := vo.ParseID(input.ID)
	if err != nil {
		return nil, err
	}

	// 1. Buscar user existente
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Aplicar atualizações parciais
	if input.Name != nil {
		e.UpdateName(*input.Name)
	}

	if input.Email != nil {
		emailVO, err := vo.NewEmail(*input.Email)
		if err != nil {
			return nil, err
		}
		e.UpdateEmail(emailVO)
	}

	// 3. Persistir alterações
	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, err
	}

	// 4. Invalidar cache
	if uc.cache != nil {
		cacheKey := "user:" + input.ID
		if err := uc.cache.Delete(ctx, cacheKey); err != nil {
			slog.Warn("failed to invalidate cache", "key", cacheKey, "error", err)
		}
	}

	return &dto.UpdateOutput{
		ID:        e.ID.String(),
		Name:      e.Name,
		Email:     e.Email.String(),
		Active:    e.Active,
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
	}, nil
}

package user

import (
	"context"
	"log/slog"
	"time"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/cache"
)

// GetUseCase implementa o caso de uso de buscar user por ID.
type GetUseCase struct {
	repo   interfaces.Repository
	cache  interfaces.Cache   // optional, set via WithCache()
	flight *cache.FlightGroup // optional, prevents cache stampede
}

// NewGetUseCase cria uma nova instância do GetUseCase.
func NewGetUseCase(repo interfaces.Repository) *GetUseCase {
	return &GetUseCase{
		repo: repo,
	}
}

// WithCache sets an optional cache for the use case (builder pattern).
func (uc *GetUseCase) WithCache(c interfaces.Cache) *GetUseCase {
	uc.cache = c
	return uc
}

// WithFlight adds singleflight protection against cache stampede (thundering herd).
func (uc *GetUseCase) WithFlight(fg *cache.FlightGroup) *GetUseCase {
	uc.flight = fg
	return uc
}

// Execute busca um user pelo ID.
//
// Fluxo com cache:
//  1. Tenta buscar no cache
//  2. Se cache miss, busca no DB
//  3. Armazena no cache para próximas requisições
func (uc *GetUseCase) Execute(ctx context.Context, input dto.GetInput) (*dto.GetOutput, error) {
	// Validar e converter ID
	id, err := vo.ParseID(input.ID)
	if err != nil {
		return nil, err
	}

	cacheKey := "user:" + input.ID

	// 1. Tentar cache primeiro
	if uc.cache != nil {
		var cached dto.GetOutput
		if cacheErr := uc.cache.Get(ctx, cacheKey, &cached); cacheErr == nil {
			slog.Debug("cache hit", "key", cacheKey)
			return &cached, nil
		}
	}

	// 2. Buscar no repositório (cache miss — with singleflight if configured)
	var e *userdomain.User

	if uc.flight != nil {
		val, flightErr, _ := uc.flight.Do(input.ID, func() (any, error) {
			return uc.repo.FindByID(ctx, id)
		})
		if flightErr != nil {
			return nil, flightErr
		}
		e = val.(*userdomain.User)
	} else {
		var findErr error
		e, findErr = uc.repo.FindByID(ctx, id)
		if findErr != nil {
			return nil, findErr
		}
	}

	// 3. Converter para DTO de saída
	output := &dto.GetOutput{
		ID:        e.ID.String(),
		Name:      e.Name,
		Email:     e.Email.String(),
		Active:    e.Active,
		CreatedAt: e.CreatedAt.Format(time.RFC3339),
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
	}

	// 4. Armazenar no cache
	if uc.cache != nil {
		if err := uc.cache.Set(ctx, cacheKey, output); err != nil {
			slog.Warn("failed to cache user", "key", cacheKey, "error", err)
		}
	}

	return output, nil
}

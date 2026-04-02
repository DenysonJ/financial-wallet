package user

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
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
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.User.Delete")
	defer span.End()

	ctx = injectLogContext(ctx, "user", "delete")

	// Validar e converter ID
	id, parseErr := vo.ParseID(input.ID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "user delete failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.ID))

	// 2. Realizar soft delete
	if deleteErr := uc.repo.Delete(ctx, id); deleteErr != nil {
		span.SetStatus(otelcodes.Error, deleteErr.Error())
		logutil.LogError(ctx, "user delete failed: repository error", "error", deleteErr.Error())
		return nil, deleteErr
	}

	// 3. Invalidar cache
	if uc.cache != nil {
		cacheKey := "user:" + input.ID
		if cacheErr := uc.cache.Delete(ctx, cacheKey); cacheErr != nil {
			logutil.LogWarn(ctx, "failed to invalidate cache", "key", cacheKey, "error", cacheErr.Error())
		}
	}

	logutil.LogInfo(ctx, "user deleted", "user.id", input.ID)

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

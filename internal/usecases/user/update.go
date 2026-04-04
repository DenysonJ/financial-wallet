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
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.User.Update")
	defer span.End()

	ctx = injectLogContext(ctx, "update")

	// Validar e converter ID
	id, parseErr := vo.ParseID(input.ID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "user update failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.ID))

	// 1. Buscar user existente
	e, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "user update failed", "error", findErr.Error())
		return nil, findErr
	}

	// 2. Aplicar atualizações parciais
	if input.Name != nil {
		e.UpdateName(*input.Name)
	}

	if input.Email != nil {
		emailVO, emailErr := vo.NewEmail(*input.Email)
		if emailErr != nil {
			span.SetStatus(otelcodes.Error, emailErr.Error())
			logutil.LogWarn(ctx, "user update failed: invalid email", "error", emailErr.Error())
			return nil, emailErr
		}
		e.UpdateEmail(emailVO)
	}

	// 3. Persistir alterações
	if updateErr := uc.repo.Update(ctx, e); updateErr != nil {
		span.SetStatus(otelcodes.Error, updateErr.Error())
		logutil.LogError(ctx, "user update failed: repository error", "error", updateErr.Error())
		return nil, updateErr
	}

	// 4. Invalidar cache
	if uc.cache != nil {
		cacheKey := "user:" + input.ID
		if cacheErr := uc.cache.Delete(ctx, cacheKey); cacheErr != nil {
			logutil.LogWarn(ctx, "failed to invalidate cache", "key", cacheKey, "error", cacheErr.Error())
		}
	}

	logutil.LogInfo(ctx, "user updated", "user.id", e.ID.String())

	return &dto.UpdateOutput{
		ID:        e.ID.String(),
		Name:      e.Name,
		Email:     e.Email.String(),
		Active:    e.Active,
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
	}, nil
}

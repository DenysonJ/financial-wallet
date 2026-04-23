package user

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
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
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.User.Update")
	defer span.End()

	ctx = injectLogContext(ctx, "update")

	id, parseErr := vo.ParseID(input.ID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		logutil.LogWarn(ctx, "user update failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.ID))

	// 1. Buscar user existente
	e, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "user update failed")
		return nil, findErr
	}

	// 2. Aplicar atualizações parciais
	if input.Name != nil {
		e.UpdateName(*input.Name)
	}

	if input.Email != nil {
		emailVO, emailErr := vo.NewEmail(*input.Email)
		if emailErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_email"))
			logutil.LogWarn(ctx, "user update failed: invalid email", "error", emailErr.Error())
			return nil, emailErr
		}
		e.UpdateEmail(emailVO)
	}

	// 3. Persistir alterações
	if updateErr := uc.repo.Update(ctx, e); updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "user update failed")
		return nil, updateErr
	}

	// 4. Invalidar cache
	if uc.cache != nil {
		cacheKey := "user:" + input.ID
		if cacheErr := uc.cache.Delete(ctx, cacheKey); cacheErr != nil {
			logutil.LogWarn(ctx, "failed to invalidate cache", "key", cacheKey, "error", cacheErr.Error())
		}
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "user updated", "user.id", e.ID.String())

	return &dto.UpdateOutput{
		ID:        e.ID.String(),
		Name:      e.Name,
		Email:     e.Email.String(),
		Active:    e.Active,
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
	}, nil
}

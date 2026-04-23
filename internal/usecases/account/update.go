package account

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// UpdateUseCase implementa o caso de uso de atualização de account.
type UpdateUseCase struct {
	repo interfaces.Repository
}

// NewUpdateUseCase cria uma nova instância do UpdateUseCase.
func NewUpdateUseCase(repo interfaces.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

// Execute atualiza uma account existente (partial update de name/description).
func (uc *UpdateUseCase) Execute(ctx context.Context, input dto.UpdateInput) (*dto.UpdateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Account.Update")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionUpdate)

	id, parseErr := uservo.ParseID(input.ID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		logutil.LogWarn(ctx, "account update failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.ID))

	// Buscar account existente
	a, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "account update failed")
		return nil, findErr
	}

	// Ownership check
	if input.RequestingUserID != "" && a.UserID.String() != input.RequestingUserID {
		telemetry.WarnSpan(span, attribute.String("app.result", "forbidden"))
		logutil.LogWarn(ctx, "account update forbidden: not owner", "account.id", a.ID.String())
		return nil, accountdomain.ErrAccountNotFound
	}

	// Aplicar atualizações parciais
	if input.Name != nil {
		a.UpdateName(*input.Name)
	}
	if input.Description != nil {
		a.UpdateDescription(*input.Description)
	}

	// Persistir
	if updateErr := uc.repo.Update(ctx, a); updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "account update failed")
		return nil, updateErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "account updated", "account.id", a.ID.String())

	return &dto.UpdateOutput{
		ID:          a.ID.String(),
		UserID:      a.UserID.String(),
		Name:        a.Name,
		Type:        a.Type.String(),
		Description: a.Description,
		Active:      a.Active,
		UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
	}, nil
}

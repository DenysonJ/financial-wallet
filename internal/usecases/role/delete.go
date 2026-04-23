package role

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// DeleteUseCase implementa o caso de uso de delecao de role.
type DeleteUseCase struct {
	repo interfaces.Repository
}

// NewDeleteUseCase cria uma nova instancia do DeleteUseCase.
func NewDeleteUseCase(repo interfaces.Repository) *DeleteUseCase {
	return &DeleteUseCase{
		repo: repo,
	}
}

// Execute realiza a delecao de uma role.
//
// Fluxo:
//  1. Validar ID
//  2. Deletar role
func (uc *DeleteUseCase) Execute(ctx context.Context, input dto.DeleteInput) (*dto.DeleteOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Role.Delete")
	defer span.End()

	ctx = injectLogContext(ctx, "delete")

	id, parseErr := vo.ParseID(input.ID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		logutil.LogWarn(ctx, "role delete failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("role.id", input.ID))

	// Deletar role
	if deleteErr := uc.repo.Delete(ctx, id); deleteErr != nil {
		telemetry.ClassifyError(ctx, span, deleteErr, "domain_error", "role delete failed")
		return nil, deleteErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "role deleted", "role.id", input.ID)

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

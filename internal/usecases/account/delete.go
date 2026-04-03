package account

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// DeleteUseCase implementa o caso de uso de deleção (soft delete) de account.
type DeleteUseCase struct {
	repo interfaces.Repository
}

// NewDeleteUseCase cria uma nova instância do DeleteUseCase.
func NewDeleteUseCase(repo interfaces.Repository) *DeleteUseCase {
	return &DeleteUseCase{repo: repo}
}

// Execute realiza soft delete de uma account.
func (uc *DeleteUseCase) Execute(ctx context.Context, input dto.DeleteInput) (*dto.DeleteOutput, error) {
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.Account.Delete")
	defer span.End()

	ctx = injectLogContext(ctx, "account", "delete")

	// Validar ID
	id, parseErr := uservo.ParseID(input.ID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "account delete failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.ID))

	// Soft delete
	if deleteErr := uc.repo.Delete(ctx, id); deleteErr != nil {
		span.SetStatus(otelcodes.Error, deleteErr.Error())
		logutil.LogError(ctx, "account delete failed: repository error", "error", deleteErr.Error())
		return nil, deleteErr
	}

	logutil.LogInfo(ctx, "account deleted", "account.id", input.ID)

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

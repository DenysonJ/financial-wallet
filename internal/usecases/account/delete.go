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
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Account.Delete")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionDelete)

	id, parseErr := uservo.ParseID(input.ID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		logutil.LogWarn(ctx, "account delete failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.ID))

	// Ownership check: fetch account first, verify owner
	if input.RequestingUserID != "" {
		a, findErr := uc.repo.FindByID(ctx, id)
		if findErr != nil {
			telemetry.ClassifyError(ctx, span, findErr, "not_found", "account delete failed")
			return nil, findErr
		}
		if a.UserID.String() != input.RequestingUserID {
			telemetry.WarnSpan(span, attribute.String("app.result", "forbidden"))
			logutil.LogWarn(ctx, "account delete forbidden: not owner", "account.id", input.ID)
			return nil, accountdomain.ErrAccountNotFound
		}
	}

	// Soft delete
	if deleteErr := uc.repo.Delete(ctx, id); deleteErr != nil {
		telemetry.ClassifyError(ctx, span, deleteErr, "domain_error", "account delete failed")
		return nil, deleteErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "account deleted", "account.id", input.ID)

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

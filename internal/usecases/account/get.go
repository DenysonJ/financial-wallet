package account

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// GetUseCase implementa o caso de uso de buscar account por ID.
type GetUseCase struct {
	repo interfaces.Repository
}

// NewGetUseCase cria uma nova instância do GetUseCase.
func NewGetUseCase(repo interfaces.Repository) *GetUseCase {
	return &GetUseCase{repo: repo}
}

// Execute busca uma account pelo ID.
func (uc *GetUseCase) Execute(ctx context.Context, input dto.GetInput) (*dto.GetOutput, error) {
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.Account.Get")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionGet)

	// Validar ID
	id, parseErr := uservo.ParseID(input.ID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "account get failed: invalid ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.ID))

	// Buscar no repositório
	a, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "account get failed", "error", findErr.Error())
		return nil, findErr
	}

	// Ownership check (when RequestingUserID is set, enforce it)
	if input.RequestingUserID != "" && a.UserID.String() != input.RequestingUserID {
		span.SetStatus(otelcodes.Error, "forbidden")
		logutil.LogWarn(ctx, "account get forbidden: not owner", "account.id", a.ID.String())
		return nil, accountdomain.ErrAccountNotFound
	}

	logutil.LogInfo(ctx, "account retrieved", "account.id", a.ID.String())

	return &dto.GetOutput{
		ID:          a.ID.String(),
		UserID:      a.UserID.String(),
		Name:        a.Name,
		Type:        a.Type.String(),
		Description: a.Description,
		Active:      a.Active,
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
	}, nil
}

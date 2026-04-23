package account

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// CreateUseCase implementa o caso de uso de criação de account.
type CreateUseCase struct {
	repo interfaces.Repository
}

// NewCreateUseCase cria uma nova instância do CreateUseCase.
func NewCreateUseCase(repo interfaces.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

// Execute executa o caso de uso de criação de account.
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.CreateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Account.Create")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionCreate)

	// Validar UserID
	userID, parseErr := uservo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		logutil.LogWarn(ctx, "account creation failed: invalid user ID", "error", parseErr.Error())
		return nil, parseErr
	}

	// Validar AccountType
	accountType, typeErr := accountvo.NewAccountType(input.Type)
	if typeErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_type"))
		logutil.LogWarn(ctx, "account creation failed: invalid type", "error", typeErr.Error())
		return nil, typeErr
	}

	// Criar entidade
	a := accountdomain.NewAccount(userID, input.Name, accountType, input.Description)

	// Persistir
	if createErr := uc.repo.Create(ctx, a); createErr != nil {
		telemetry.ClassifyError(ctx, span, createErr, "domain_error", "account creation failed")
		return nil, createErr
	}

	span.SetAttributes(attribute.String("account.id", a.ID.String()))
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "account created", "account.id", a.ID.String())

	return &dto.CreateOutput{
		ID:        a.ID.String(),
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}, nil
}

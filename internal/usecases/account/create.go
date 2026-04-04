package account

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
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
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.Account.Create")
	defer span.End()

	ctx = injectLogContext(ctx, resourceAccount, logutil.ActionCreate)

	// Validar UserID
	userID, parseErr := uservo.ParseID(input.UserID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "account creation failed: invalid user ID", "error", parseErr.Error())
		return nil, parseErr
	}

	// Validar AccountType
	accountType, typeErr := accountvo.NewAccountType(input.Type)
	if typeErr != nil {
		span.SetStatus(otelcodes.Error, typeErr.Error())
		logutil.LogWarn(ctx, "account creation failed: invalid type", "error", typeErr.Error())
		return nil, typeErr
	}

	// Criar entidade
	a := accountdomain.NewAccount(userID, input.Name, accountType, input.Description)

	// Persistir
	if createErr := uc.repo.Create(ctx, a); createErr != nil {
		span.SetStatus(otelcodes.Error, createErr.Error())
		logutil.LogError(ctx, "account creation failed: repository error", "error", createErr.Error())
		return nil, createErr
	}

	span.SetAttributes(attribute.String("account.id", a.ID.String()))
	logutil.LogInfo(ctx, "account created", "account.id", a.ID.String())

	return &dto.CreateOutput{
		ID:        a.ID.String(),
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}, nil
}

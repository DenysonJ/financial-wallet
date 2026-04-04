package user

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// CreateUseCase implementa o caso de uso de criação de user.
type CreateUseCase struct {
	repo interfaces.Repository
}

// NewCreateUseCase cria uma nova instância do CreateUseCase.
func NewCreateUseCase(repo interfaces.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

// Execute executa o caso de uso de criação de user.
//
// Fluxo:
//  1. Converte primitivos (string) para Value Objects (validação acontece aqui)
//  2. Cria a entidade User usando a Factory
//  3. Persiste no banco via Repository
//  4. Retorna DTO com ID e timestamp
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.CreateOutput, error) {
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.User.Create")
	defer span.End()

	ctx = injectLogContext(ctx, "create")

	// PASSO 1: Converter primitivos para Value Objects
	emailVO, emailErr := vo.NewEmail(input.Email)
	if emailErr != nil {
		span.SetStatus(otelcodes.Error, emailErr.Error())
		logutil.LogWarn(ctx, "user creation failed: invalid email", "error", emailErr.Error())
		return nil, emailErr
	}

	// PASSO 2: Criar Entidade usando a Factory
	e := userdomain.NewUser(input.Name, emailVO)

	// PASSO 3: Persistir no banco via Repository
	if createErr := uc.repo.Create(ctx, e); createErr != nil {
		span.SetStatus(otelcodes.Error, createErr.Error())
		logutil.LogError(ctx, "user creation failed: repository error", "error", createErr.Error())
		return nil, createErr
	}

	// PASSO 4: Retornar Output DTO
	span.SetAttributes(attribute.String("user.id", e.ID.String()))
	logutil.LogInfo(ctx, "user created", "user.id", e.ID.String())

	return &dto.CreateOutput{
		ID:        e.ID.String(),
		CreatedAt: e.CreatedAt.Format(time.RFC3339),
	}, nil
}

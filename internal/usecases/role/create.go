package role

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// CreateUseCase implementa o caso de uso de criacao de role.
type CreateUseCase struct {
	repo interfaces.Repository
}

// NewCreateUseCase cria uma nova instancia do CreateUseCase.
func NewCreateUseCase(repo interfaces.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

// Execute executa o caso de uso de criacao de role.
//
// Fluxo:
//  1. Verifica se ja existe uma role com o mesmo nome
//  2. Cria a entidade Role usando a Factory
//  3. Persiste no banco via Repository
//  4. Retorna DTO com ID e timestamp
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.CreateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Role.Create")
	defer span.End()

	ctx = injectLogContext(ctx, "create")

	// PASSO 1: Verificar duplicidade de nome
	existingRole, findErr := uc.repo.FindByName(ctx, input.Name)
	if findErr != nil && !errors.Is(findErr, roledomain.ErrRoleNotFound) {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogError(ctx, "role creation failed: repository error", "error", findErr.Error())
		return nil, findErr
	}
	if existingRole != nil {
		span.SetStatus(otelcodes.Error, roledomain.ErrDuplicateRoleName.Error())
		logutil.LogWarn(ctx, "role creation failed: duplicate name", "role.name", input.Name)
		return nil, roledomain.ErrDuplicateRoleName
	}

	// PASSO 2: Criar Entidade usando a Factory
	r := roledomain.NewRole(input.Name, input.Description)

	// PASSO 3: Persistir no banco via Repository
	if createErr := uc.repo.Create(ctx, r); createErr != nil {
		span.SetStatus(otelcodes.Error, createErr.Error())
		logutil.LogError(ctx, "role creation failed: repository error", "error", createErr.Error())
		return nil, createErr
	}

	// PASSO 4: Retornar Output DTO
	span.SetAttributes(attribute.String("role.id", r.ID.String()))
	logutil.LogInfo(ctx, "role created", "role.id", r.ID.String())

	return &dto.CreateOutput{
		ID:        r.ID.String(),
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}, nil
}

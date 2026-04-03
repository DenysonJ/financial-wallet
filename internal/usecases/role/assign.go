package role

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/cache"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// AssignRoleUseCase implementa o caso de uso de atribuir uma role a um usuário.
type AssignRoleUseCase struct {
	repo  interfaces.Repository
	cache cache.Cache
}

// NewAssignRoleUseCase cria uma nova instância do AssignRoleUseCase.
func NewAssignRoleUseCase(repo interfaces.Repository) *AssignRoleUseCase {
	return &AssignRoleUseCase{repo: repo}
}

// WithCache sets an optional cache for permission invalidation (builder pattern).
func (uc *AssignRoleUseCase) WithCache(c cache.Cache) *AssignRoleUseCase {
	uc.cache = c
	return uc
}

// Execute atribui uma role a um usuário.
//
// Fluxo:
//  1. Validar UserID e RoleID
//  2. Verificar se a role existe
//  3. Atribuir role ao usuário
//  4. Invalidar cache de permissions do usuário
func (uc *AssignRoleUseCase) Execute(ctx context.Context, input dto.AssignRoleInput) error {
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.Role.Assign")
	defer span.End()

	ctx = injectLogContext(ctx, "role", "assign")

	userID, userParseErr := vo.ParseID(input.UserID)
	if userParseErr != nil {
		span.SetStatus(otelcodes.Error, userParseErr.Error())
		logutil.LogWarn(ctx, "role assign failed: invalid user ID", "error", userParseErr.Error())
		return userParseErr
	}

	roleID, roleParseErr := vo.ParseID(input.RoleID)
	if roleParseErr != nil {
		span.SetStatus(otelcodes.Error, roleParseErr.Error())
		logutil.LogWarn(ctx, "role assign failed: invalid role ID", "error", roleParseErr.Error())
		return roleParseErr
	}

	span.SetAttributes(
		attribute.String("user.id", input.UserID),
		attribute.String("role.id", input.RoleID),
	)

	// Verify role exists
	_, findErr := uc.repo.FindByID(ctx, roleID)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "role assign failed: role not found", "error", findErr.Error())
		return findErr
	}

	// Assign role
	assignErr := uc.repo.AssignRole(ctx, userID, roleID)
	if assignErr != nil {
		span.SetStatus(otelcodes.Error, assignErr.Error())
		logutil.LogWarn(ctx, "role assign failed", "error", assignErr.Error())
		return assignErr
	}

	// Invalidate permissions cache
	if uc.cache != nil {
		cacheKey := interfaces.PermissionCacheKeyPrefix + input.UserID
		if cacheErr := uc.cache.Delete(ctx, cacheKey); cacheErr != nil {
			logutil.LogWarn(ctx, "failed to invalidate permissions cache", "key", cacheKey, "error", cacheErr.Error())
		}
	}

	logutil.LogInfo(ctx, "role assigned", "user.id", input.UserID, "role.id", input.RoleID)

	return nil
}

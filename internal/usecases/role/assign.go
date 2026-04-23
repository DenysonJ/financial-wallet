package role

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// AssignRoleUseCase implementa o caso de uso de atribuir uma role a um usuário.
type AssignRoleUseCase struct {
	repo  interfaces.Repository
	cache interfaces.Cache
}

// NewAssignRoleUseCase cria uma nova instância do AssignRoleUseCase.
func NewAssignRoleUseCase(repo interfaces.Repository) *AssignRoleUseCase {
	return &AssignRoleUseCase{repo: repo}
}

// WithCache sets an optional cache for permission invalidation (builder pattern).
func (uc *AssignRoleUseCase) WithCache(c interfaces.Cache) *AssignRoleUseCase {
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
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Role.Assign")
	defer span.End()

	ctx = injectLogContext(ctx, "assign")

	userID, userParseErr := vo.ParseID(input.UserID)
	if userParseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		logutil.LogWarn(ctx, "role assign failed: invalid user ID", "error", userParseErr.Error())
		return userParseErr
	}

	roleID, roleParseErr := vo.ParseID(input.RoleID)
	if roleParseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_role_id"))
		logutil.LogWarn(ctx, "role assign failed: invalid role ID", "error", roleParseErr.Error())
		return roleParseErr
	}

	span.SetAttributes(
		attribute.String("user.id", input.UserID),
		attribute.String("role.id", input.RoleID),
	)

	// Verify role exists
	if _, findErr := uc.repo.FindByID(ctx, roleID); findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "role assign failed")
		return findErr
	}

	// Assign role
	if assignErr := uc.repo.AssignRole(ctx, userID, roleID); assignErr != nil {
		telemetry.ClassifyError(ctx, span, assignErr, "domain_error", "role assign failed")
		return assignErr
	}

	// Invalidate permissions cache
	if uc.cache != nil {
		cacheKey := interfaces.PermissionCacheKeyPrefix + input.UserID
		if cacheErr := uc.cache.Delete(ctx, cacheKey); cacheErr != nil {
			logutil.LogWarn(ctx, "failed to invalidate permissions cache", "key", cacheKey, "error", cacheErr.Error())
		}
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "role assigned", "user.id", input.UserID, "role.id", input.RoleID)

	return nil
}

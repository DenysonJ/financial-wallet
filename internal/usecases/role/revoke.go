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

// RevokeRoleUseCase implementa o caso de uso de revogar uma role de um usuário.
type RevokeRoleUseCase struct {
	repo  interfaces.Repository
	cache cache.Cache
}

// NewRevokeRoleUseCase cria uma nova instância do RevokeRoleUseCase.
func NewRevokeRoleUseCase(repo interfaces.Repository) *RevokeRoleUseCase {
	return &RevokeRoleUseCase{repo: repo}
}

// WithCache sets an optional cache for permission invalidation (builder pattern).
func (uc *RevokeRoleUseCase) WithCache(c cache.Cache) *RevokeRoleUseCase {
	uc.cache = c
	return uc
}

// Execute revoga uma role de um usuário.
//
// Fluxo:
//  1. Validar UserID e RoleID
//  2. Revogar role do usuário
//  3. Invalidar cache de permissions do usuário
func (uc *RevokeRoleUseCase) Execute(ctx context.Context, input dto.RevokeRoleInput) error {
	ctx, span := otel.Tracer("usecase").Start(ctx, "UseCase.Role.Revoke")
	defer span.End()

	ctx = injectLogContext(ctx, "role", "revoke")

	userID, userParseErr := vo.ParseID(input.UserID)
	if userParseErr != nil {
		span.SetStatus(otelcodes.Error, userParseErr.Error())
		logutil.LogWarn(ctx, "role revoke failed: invalid user ID", "error", userParseErr.Error())
		return userParseErr
	}

	roleID, roleParseErr := vo.ParseID(input.RoleID)
	if roleParseErr != nil {
		span.SetStatus(otelcodes.Error, roleParseErr.Error())
		logutil.LogWarn(ctx, "role revoke failed: invalid role ID", "error", roleParseErr.Error())
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
		logutil.LogWarn(ctx, "role revoke failed: role not found", "error", findErr.Error())
		return findErr
	}

	// Revoke role
	revokeErr := uc.repo.RevokeRole(ctx, userID, roleID)
	if revokeErr != nil {
		span.SetStatus(otelcodes.Error, revokeErr.Error())
		logutil.LogWarn(ctx, "role revoke failed", "error", revokeErr.Error())
		return revokeErr
	}

	// Invalidate permissions cache
	if uc.cache != nil {
		cacheKey := "permissions:user:" + input.UserID
		if cacheErr := uc.cache.Delete(ctx, cacheKey); cacheErr != nil {
			logutil.LogWarn(ctx, "failed to invalidate permissions cache", "key", cacheKey, "error", cacheErr.Error())
		}
	}

	logutil.LogInfo(ctx, "role revoked", "user.id", input.UserID, "role.id", input.RoleID)

	return nil
}

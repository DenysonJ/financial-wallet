package role

import (
	"context"

	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// invalidateUserAuthCache deletes the user's permissions and roles cache
// entries. Both must be invalidated together: leaving the roles entry stale
// would let admin checks pass via GetRoles even after the permissions cache
// was already rotated. Cache failures are logged and swallowed — the source
// of truth has already been updated by the caller.
func invalidateUserAuthCache(ctx context.Context, cache interfaces.Cache, userID string) {
	if cache == nil {
		return
	}

	permsKey := interfaces.PermissionCacheKeyPrefix + userID
	if cacheErr := cache.Delete(ctx, permsKey); cacheErr != nil {
		logutil.LogWarn(ctx, "failed to invalidate permissions cache", "key", permsKey, "error", cacheErr.Error())
	}

	rolesKey := interfaces.RoleCacheKeyPrefix + userID
	if cacheErr := cache.Delete(ctx, rolesKey); cacheErr != nil {
		logutil.LogWarn(ctx, "failed to invalidate roles cache", "key", rolesKey, "error", cacheErr.Error())
	}
}

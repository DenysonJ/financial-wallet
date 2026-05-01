package interfaces

// PermissionCacheKeyPrefix is the Redis key prefix for user permissions cache.
// Used by both the use case layer (cache invalidation) and infrastructure layer (cache loading).
const PermissionCacheKeyPrefix = "permissions:user:"

// RoleCacheKeyPrefix is the Redis key prefix for user roles cache.
// Both perms and roles caches must be invalidated together on assign/revoke,
// otherwise an admin role could still pass RBAC checks via the stale roles
// cache after the permissions cache was already cleared.
const RoleCacheKeyPrefix = "roles:user:"

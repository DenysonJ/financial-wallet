package interfaces

// PermissionCacheKeyPrefix is the Redis key prefix for user permissions cache.
// Used by both the use case layer (cache invalidation) and infrastructure layer (cache loading).
const PermissionCacheKeyPrefix = "permissions:user:"

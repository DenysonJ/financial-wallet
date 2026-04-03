package middleware

import (
	"context"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/cache"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

// PermissionRepository defines the contract for loading user permissions from the database.
type PermissionRepository interface {
	GetUserPermissions(ctx context.Context, userID vo.ID) ([]string, error)
}

// CachedPermissionLoader loads permissions with Redis cache and DB fallback.
type CachedPermissionLoader struct {
	repo  PermissionRepository
	cache cache.Cache
}

// NewCachedPermissionLoader creates a new CachedPermissionLoader.
func NewCachedPermissionLoader(repo PermissionRepository, c cache.Cache) *CachedPermissionLoader {
	return &CachedPermissionLoader{repo: repo, cache: c}
}

// GetPermissions returns the user's permissions, checking cache first.
func (l *CachedPermissionLoader) GetPermissions(ctx context.Context, userID string) ([]string, error) {
	cacheKey := interfaces.PermissionCacheKeyPrefix + userID

	// 1. Try cache
	if l.cache != nil {
		var cached []string
		if cacheErr := l.cache.Get(ctx, cacheKey, &cached); cacheErr == nil {
			return cached, nil
		}
	}

	// 2. Fallback to DB
	id, parseErr := vo.ParseID(userID)
	if parseErr != nil {
		return nil, parseErr
	}

	permissions, dbErr := l.repo.GetUserPermissions(ctx, id)
	if dbErr != nil {
		return nil, dbErr
	}

	// 3. Cache for next time
	if l.cache != nil {
		if setCacheErr := l.cache.Set(ctx, cacheKey, permissions); setCacheErr != nil {
			logutil.LogWarn(ctx, "failed to cache permissions", "key", cacheKey, "error", setCacheErr.Error())
		}
	}

	return permissions, nil
}

package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/mocks/middlewaremock"
	"github.com/DenysonJ/financial-wallet/internal/mocks/useruci"
	"github.com/DenysonJ/financial-wallet/pkg/cache"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCachedPermissionLoader_GetPermissions(t *testing.T) {
	validID := uuid.Must(uuid.NewV7()).String()

	tests := []struct {
		name       string
		userID     string
		setupCache func(c *useruci.MockCache)
		setupRepo  func(r *middlewaremock.MockPermissionRepository)
		wantPerms  []string
		wantErr    bool
		noCache    bool
	}{
		{
			name:   "cache hit returns cached permissions",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, "permissions:user:"+validID, mock.Anything).
					Run(func(args mock.Arguments) {
						dest := args.Get(2).(*[]string)
						*dest = []string{"user:read", "user:write"}
					}).Return(nil)
			},
			setupRepo: func(_ *middlewaremock.MockPermissionRepository) {},
			wantPerms: []string{"user:read", "user:write"},
		},
		{
			name:   "cache miss queries DB and caches result",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, "permissions:user:"+validID, mock.Anything).
					Return(cache.ErrCacheMiss)
				c.On("Set", mock.Anything, "permissions:user:"+validID, []string{"user:read"}).
					Return(nil)
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return([]string{"user:read"}, nil)
			},
			wantPerms: []string{"user:read"},
		},
		{
			name:   "DB error propagated",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, mock.Anything, mock.Anything).
					Return(cache.ErrCacheMiss)
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return(nil, errors.New("db connection failed"))
			},
			wantErr: true,
		},
		{
			name:   "repo error on invalid user ID propagated",
			userID: "not-a-uuid",
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, mock.Anything, mock.Anything).
					Return(cache.ErrCacheMiss)
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserPermissions", mock.Anything, "not-a-uuid").
					Return(nil, errors.New("invalid ID"))
			},
			wantErr: true,
		},
		{
			name:       "nil cache queries DB directly",
			userID:     validID,
			noCache:    true,
			setupCache: func(_ *useruci.MockCache) {},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return([]string{"role:read"}, nil)
			},
			wantPerms: []string{"role:read"},
		},
		{
			name:   "cache set error still returns permissions",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, "permissions:user:"+validID, mock.Anything).
					Return(cache.ErrCacheMiss)
				c.On("Set", mock.Anything, "permissions:user:"+validID, []string{"user:read"}).
					Return(errors.New("redis connection lost"))
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return([]string{"user:read"}, nil)
			},
			wantPerms: []string{"user:read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := middlewaremock.NewMockPermissionRepository(t)
			mockCacheInst := useruci.NewMockCache(t)

			tt.setupCache(mockCacheInst)
			tt.setupRepo(mockRepo)

			var loader *CachedPermissionLoader
			if tt.noCache {
				loader = NewCachedPermissionLoader(mockRepo, nil)
			} else {
				loader = NewCachedPermissionLoader(mockRepo, mockCacheInst)
			}

			perms, permErr := loader.GetPermissions(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, permErr)
				assert.Nil(t, perms)
			} else {
				assert.NoError(t, permErr)
				assert.Equal(t, tt.wantPerms, perms)
			}

			mockRepo.AssertExpectations(t)
			if !tt.noCache {
				mockCacheInst.AssertExpectations(t)
			}
		})
	}
}

func TestCachedPermissionLoader_GetRoles(t *testing.T) {
	validID := uuid.Must(uuid.NewV7()).String()

	tests := []struct {
		name       string
		userID     string
		setupCache func(c *useruci.MockCache)
		setupRepo  func(r *middlewaremock.MockPermissionRepository)
		wantRoles  []string
		wantErr    bool
		noCache    bool
	}{
		{
			name:   "cache hit retorna roles cacheadas",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, "roles:user:"+validID, mock.Anything).
					Run(func(args mock.Arguments) {
						dest := args.Get(2).(*[]string)
						*dest = []string{"admin", "user"}
					}).Return(nil)
			},
			setupRepo: func(_ *middlewaremock.MockPermissionRepository) {},
			wantRoles: []string{"admin", "user"},
		},
		{
			name:   "cache miss busca no DB e faz cache",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, "roles:user:"+validID, mock.Anything).
					Return(cache.ErrCacheMiss)
				c.On("Set", mock.Anything, "roles:user:"+validID, []string{"user"}).
					Return(nil)
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserRoles", mock.Anything, validID).
					Return([]string{"user"}, nil)
			},
			wantRoles: []string{"user"},
		},
		{
			name:   "erro do DB propagado",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, mock.Anything, mock.Anything).
					Return(cache.ErrCacheMiss)
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserRoles", mock.Anything, validID).
					Return(nil, errors.New("db connection failed"))
			},
			wantErr: true,
		},
		{
			name:       "sem cache busca direto no DB",
			userID:     validID,
			noCache:    true,
			setupCache: func(_ *useruci.MockCache) {},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserRoles", mock.Anything, validID).
					Return([]string{"admin"}, nil)
			},
			wantRoles: []string{"admin"},
		},
		{
			name:   "erro no cache set ainda retorna roles",
			userID: validID,
			setupCache: func(c *useruci.MockCache) {
				c.On("Get", mock.Anything, "roles:user:"+validID, mock.Anything).
					Return(cache.ErrCacheMiss)
				c.On("Set", mock.Anything, "roles:user:"+validID, []string{"user"}).
					Return(errors.New("redis connection lost"))
			},
			setupRepo: func(r *middlewaremock.MockPermissionRepository) {
				r.On("GetUserRoles", mock.Anything, validID).
					Return([]string{"user"}, nil)
			},
			wantRoles: []string{"user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := middlewaremock.NewMockPermissionRepository(t)
			mockCacheInst := useruci.NewMockCache(t)

			tt.setupCache(mockCacheInst)
			tt.setupRepo(mockRepo)

			var loader *CachedPermissionLoader
			if tt.noCache {
				loader = NewCachedPermissionLoader(mockRepo, nil)
			} else {
				loader = NewCachedPermissionLoader(mockRepo, mockCacheInst)
			}

			roles, rolesErr := loader.GetRoles(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, rolesErr)
				assert.Nil(t, roles)
			} else {
				assert.NoError(t, rolesErr)
				assert.Equal(t, tt.wantRoles, roles)
			}

			mockRepo.AssertExpectations(t)
			if !tt.noCache {
				mockCacheInst.AssertExpectations(t)
			}
		})
	}
}

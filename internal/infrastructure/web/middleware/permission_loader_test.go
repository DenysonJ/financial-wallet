package middleware

import (
	"context"
	"errors"
	"testing"

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
		setupCache func(c *mockCache)
		setupRepo  func(r *mockPermissionRepo)
		wantPerms  []string
		wantErr    bool
		noCache    bool
	}{
		{
			name:   "cache hit returns cached permissions",
			userID: validID,
			setupCache: func(c *mockCache) {
				c.On("Get", mock.Anything, "permissions:user:"+validID, mock.Anything).
					Run(func(args mock.Arguments) {
						dest := args.Get(2).(*[]string)
						*dest = []string{"user:read", "user:write"}
					}).Return(nil)
			},
			setupRepo: func(_ *mockPermissionRepo) {},
			wantPerms: []string{"user:read", "user:write"},
		},
		{
			name:   "cache miss queries DB and caches result",
			userID: validID,
			setupCache: func(c *mockCache) {
				c.On("Get", mock.Anything, "permissions:user:"+validID, mock.Anything).
					Return(cache.ErrCacheMiss)
				c.On("Set", mock.Anything, "permissions:user:"+validID, []string{"user:read"}).
					Return(nil)
			},
			setupRepo: func(r *mockPermissionRepo) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return([]string{"user:read"}, nil)
			},
			wantPerms: []string{"user:read"},
		},
		{
			name:   "DB error propagated",
			userID: validID,
			setupCache: func(c *mockCache) {
				c.On("Get", mock.Anything, mock.Anything, mock.Anything).
					Return(cache.ErrCacheMiss)
			},
			setupRepo: func(r *mockPermissionRepo) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return(nil, errors.New("db connection failed"))
			},
			wantErr: true,
		},
		{
			name:   "repo error on invalid user ID propagated",
			userID: "not-a-uuid",
			setupCache: func(c *mockCache) {
				c.On("Get", mock.Anything, mock.Anything, mock.Anything).
					Return(cache.ErrCacheMiss)
			},
			setupRepo: func(r *mockPermissionRepo) {
				r.On("GetUserPermissions", mock.Anything, "not-a-uuid").
					Return(nil, errors.New("invalid ID"))
			},
			wantErr: true,
		},
		{
			name:       "nil cache queries DB directly",
			userID:     validID,
			noCache:    true,
			setupCache: func(_ *mockCache) {},
			setupRepo: func(r *mockPermissionRepo) {
				r.On("GetUserPermissions", mock.Anything, validID).
					Return([]string{"role:read"}, nil)
			},
			wantPerms: []string{"role:read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mockPermissionRepo)
			mockCacheInst := new(mockCache)

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

package main

import (
	"context"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	roleinterfaces "github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
)

// permissionRepoAdapter adapts role/interfaces.Repository (vo.ID params) to
// middleware.PermissionRepository (string params), keeping domain VO parsing
// out of the middleware layer.
type permissionRepoAdapter struct {
	repo roleinterfaces.Repository
}

func (a *permissionRepoAdapter) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	id, parseErr := vo.ParseID(userID)
	if parseErr != nil {
		return nil, parseErr
	}
	return a.repo.GetUserPermissions(ctx, id)
}

func (a *permissionRepoAdapter) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	id, parseErr := vo.ParseID(userID)
	if parseErr != nil {
		return nil, parseErr
	}
	return a.repo.GetUserRoles(ctx, id)
}

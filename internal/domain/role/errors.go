package role

import "errors"

// Erros de domínio para Role.
var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrDuplicateRoleName = errors.New("role name already exists")
)

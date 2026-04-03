package role

import "errors"

// Erros de domínio para Role.
var (
	ErrRoleNotFound        = errors.New("role not found")
	ErrDuplicateRoleName   = errors.New("role name already exists")
	ErrRoleAlreadyAssigned = errors.New("role already assigned to user")
	ErrRoleNotAssigned     = errors.New("role not assigned to user")
	ErrForbidden           = errors.New("forbidden")
)

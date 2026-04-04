package account

import "errors"

// Erros de domínio para Account.
var (
	ErrAccountNotFound = errors.New("account not found")
	ErrForbidden       = errors.New("forbidden")
)

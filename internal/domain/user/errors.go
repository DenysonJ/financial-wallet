package user

import "errors"

// Erros de domínio para User.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrPasswordAlreadySet = errors.New("password already set")
	ErrPasswordMismatch   = errors.New("passwords do not match")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
)

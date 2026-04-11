package vo

import "errors"

var (
	ErrInvalidStatementType = errors.New("invalid statement type")
	ErrInvalidAmount        = errors.New("amount must be greater than zero")
)

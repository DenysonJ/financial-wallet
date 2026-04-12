package statement

import "errors"

var (
	ErrStatementNotFound = errors.New("statement not found")
	ErrAlreadyReversed   = errors.New("statement already reversed")
	ErrAccountNotActive  = errors.New("account is not active")
)

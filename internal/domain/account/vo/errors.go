package vo

import "errors"

var (
	ErrInvalidAccountType = errors.New("invalid account type")
	ErrInvalidCreditLimit = errors.New("invalid credit limit")
)

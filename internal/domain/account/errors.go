package account

import "errors"

// Erros de domínio para Account.
var (
	ErrAccountNotFound = errors.New("account not found")

	// ErrCreditLimitRequired: credit_card sem limite de crédito.
	ErrCreditLimitRequired = errors.New("credit limit is required")

	// ErrCreditLimitNotAllowed: limite de crédito em tipo que não o admite.
	ErrCreditLimitNotAllowed = errors.New("credit limit is not allowed for this account type")
)

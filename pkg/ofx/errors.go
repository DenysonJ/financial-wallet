package ofx

import "errors"

var (
	ErrInvalidFormat  = errors.New("invalid OFX format")
	ErrNoTransactions = errors.New("no transactions found in OFX file")
	ErrInvalidAmount  = errors.New("invalid OFX amount")
	ErrInvalidDate    = errors.New("invalid OFX date")
)

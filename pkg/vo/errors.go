package vo

import "errors"

// ErrInvalidID is returned when a string is not a valid UUID.
var ErrInvalidID = errors.New("invalid ID")

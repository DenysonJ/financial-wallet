package vo

import (
	"database/sql/driver"
	"fmt"
)

// Amount represents a monetary value in cents (always positive).
type Amount int64

// NewAmount validates and creates an Amount (must be > 0).
func NewAmount(value int64) (Amount, error) {
	if value <= 0 {
		return 0, ErrInvalidAmount
	}
	return Amount(value), nil
}

// ParseAmount creates an Amount without validation (for DB reads).
func ParseAmount(value int64) Amount {
	return Amount(value)
}

// Int64 returns the underlying int64 value.
func (a Amount) Int64() int64 {
	return int64(a)
}

// Value implements driver.Valuer for database storage.
func (a Amount) Value() (driver.Value, error) {
	return int64(a), nil
}

// Scan implements sql.Scanner for database retrieval.
func (a *Amount) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("amount cannot be nil")
	}
	switch v := value.(type) {
	case int64:
		*a = Amount(v)
	case float64:
		*a = Amount(int64(v))
	default:
		return fmt.Errorf("invalid type for Amount: %T", value)
	}
	return nil
}

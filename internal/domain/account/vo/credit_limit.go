package vo

import (
	"database/sql/driver"
	"fmt"
)

// MaxCreditLimit is the ceiling accepted for a credit limit, in cents
// (R$ 10 billion). It guards against absurd input and keeps the sum of every
// limit of a user well inside int64 range
const MaxCreditLimit int64 = 1_000_000_000_000

// CreditLimit represents the credit limit of an account, in cents.
// Always positive and never above MaxCreditLimit.
type CreditLimit int64

// NewCreditLimit validates and creates a CreditLimit (must be > 0 and <= MaxCreditLimit).
func NewCreditLimit(value int64) (CreditLimit, error) {
	if value <= 0 || value > MaxCreditLimit {
		return 0, ErrInvalidCreditLimit
	}
	return CreditLimit(value), nil
}

// ParseCreditLimit creates a CreditLimit without validation (for DB reads).
func ParseCreditLimit(value int64) CreditLimit {
	return CreditLimit(value)
}

// Int64 returns the underlying int64 value.
func (c CreditLimit) Int64() int64 {
	return int64(c)
}

// Value implements driver.Valuer for database storage.
func (c CreditLimit) Value() (driver.Value, error) {
	return int64(c), nil
}

// Scan implements sql.Scanner for database retrieval.
func (c *CreditLimit) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("credit limit cannot be nil")
	}
	switch v := value.(type) {
	case int64:
		*c = CreditLimit(v)
	case float64:
		*c = CreditLimit(int64(v))
	default:
		return fmt.Errorf("invalid type for CreditLimit: %T", value)
	}
	return nil
}

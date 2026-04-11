package vo

import (
	"database/sql/driver"
	"fmt"
)

// StatementType represents the direction of a financial statement.
type StatementType string

const (
	TypeCredit StatementType = "credit"
	TypeDebit  StatementType = "debit"
)

// validTypes contains all allowed statement types.
var validTypes = map[StatementType]bool{
	TypeCredit: true,
	TypeDebit:  true,
}

// NewStatementType validates and creates a StatementType.
func NewStatementType(value string) (StatementType, error) {
	t := StatementType(value)
	if !validTypes[t] {
		return "", ErrInvalidStatementType
	}
	return t, nil
}

// ParseStatementType creates a StatementType without validation (for DB reads).
func ParseStatementType(value string) StatementType {
	return StatementType(value)
}

// Opposite returns the reverse statement type.
func (t StatementType) Opposite() StatementType {
	if t == TypeCredit {
		return TypeDebit
	}
	return TypeCredit
}

// String returns the string representation.
func (t StatementType) String() string {
	return string(t)
}

// Value implements driver.Valuer for database storage.
func (t StatementType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan implements sql.Scanner for database retrieval.
func (t *StatementType) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("statement type cannot be nil")
	}
	switch v := value.(type) {
	case string:
		*t = StatementType(v)
	case []byte:
		*t = StatementType(string(v))
	default:
		return fmt.Errorf("invalid type for StatementType: %T", value)
	}
	return nil
}

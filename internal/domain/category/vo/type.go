package vo

import (
	"database/sql/driver"
	"fmt"
)

// CategoryType represents the financial direction of a category.
// Mirrors statement.vo.StatementType values (credit/debit) but is a
// separate type to keep domains autonomous.
type CategoryType string

const (
	TypeCredit CategoryType = "credit"
	TypeDebit  CategoryType = "debit"
)

var validTypes = map[CategoryType]bool{
	TypeCredit: true,
	TypeDebit:  true,
}

// NewCategoryType validates and creates a CategoryType.
func NewCategoryType(value string) (CategoryType, error) {
	t := CategoryType(value)
	if !validTypes[t] {
		return "", ErrInvalidCategoryType
	}
	return t, nil
}

// ParseCategoryType creates a CategoryType without validation (for DB reads).
func ParseCategoryType(value string) CategoryType {
	return CategoryType(value)
}

// String returns the string representation.
func (t CategoryType) String() string {
	return string(t)
}

// Value implements driver.Valuer for database storage.
func (t CategoryType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan implements sql.Scanner for database retrieval.
func (t *CategoryType) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("category type cannot be nil")
	}
	switch v := value.(type) {
	case string:
		*t = CategoryType(v)
	case []byte:
		*t = CategoryType(string(v))
	default:
		return fmt.Errorf("invalid type for CategoryType: %T", value)
	}
	return nil
}

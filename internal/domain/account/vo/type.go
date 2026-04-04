package vo

import (
	"database/sql/driver"
	"fmt"
)

// AccountType represents the kind of financial account.
type AccountType string

const (
	TypeBankAccount AccountType = "bank_account"
	TypeCreditCard  AccountType = "credit_card"
	TypeCash        AccountType = "cash"
)

// validTypes contains all allowed account types.
var validTypes = map[AccountType]bool{
	TypeBankAccount: true,
	TypeCreditCard:  true,
	TypeCash:        true,
}

// NewAccountType validates and creates an AccountType.
func NewAccountType(value string) (AccountType, error) {
	t := AccountType(value)
	if !validTypes[t] {
		return "", ErrInvalidAccountType
	}
	return t, nil
}

// ParseAccountType creates an AccountType without validation (for DB reads).
func ParseAccountType(value string) AccountType {
	return AccountType(value)
}

// String returns the string representation.
func (t AccountType) String() string {
	return string(t)
}

// Value implements driver.Valuer for database storage.
func (t AccountType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan implements sql.Scanner for database retrieval.
func (t *AccountType) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("account type cannot be nil")
	}
	switch v := value.(type) {
	case string:
		*t = AccountType(v)
	case []byte:
		*t = AccountType(string(v))
	default:
		return fmt.Errorf("invalid type for AccountType: %T", value)
	}
	return nil
}

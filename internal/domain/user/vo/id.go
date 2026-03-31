package vo

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ID represents a unique entity identifier using UUID v7 (RFC 9562).
// UUID v7 is time-ordered (48-bit unix timestamp + 74-bit randomness),
// providing good B-tree index locality and global uniqueness.
type ID string

// NewID generates a new UUID v7 identifier.
func NewID() ID {
	return ID(uuid.Must(uuid.NewV7()).String())
}

// ParseID validates and parses a UUID v7 string into an ID.
func ParseID(s string) (ID, error) {
	if _, parseErr := uuid.Parse(s); parseErr != nil {
		return "", ErrInvalidID
	}
	return ID(s), nil
}

// String returns the string representation.
func (i ID) String() string {
	return string(i)
}

// Value implements driver.Valuer for database storage.
func (i ID) Value() (driver.Value, error) {
	return string(i), nil
}

// Scan implements sql.Scanner for database retrieval.
func (i *ID) Scan(value any) error {
	if value == nil {
		return errors.New("ID cannot be empty")
	}
	sv, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for ID: %T", value)
	}
	if _, parseErr := uuid.Parse(sv); parseErr != nil {
		return ErrInvalidID
	}
	*i = ID(sv)
	return nil
}

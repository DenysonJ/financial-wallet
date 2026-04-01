package vo

import (
	"database/sql/driver"
	"fmt"
	"net/mail"
)

type Email struct {
	value string
}

func NewEmail(value string) (Email, error) {
	if _, err := mail.ParseAddress(value); err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: value}, nil
}

func ParseEmail(value string) Email {
	return Email{value: value}
}

func (e Email) String() string { return e.value }

func (e Email) Value() (driver.Value, error) {
	return e.value, nil
}

func (e *Email) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("email can't be nil")
	}
	switch v := value.(type) {
	case string:
		e.value = v
	case []byte:
		e.value = string(v)
	default:
		return fmt.Errorf("wrong type for Email: %T", value)
	}
	return nil
}

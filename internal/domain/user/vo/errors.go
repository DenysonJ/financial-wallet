package vo

import "errors"

// =============================================================================
// ERROS DE VALUE OBJECTS (PUROS)
// =============================================================================
//
// Estes erros são usados pelos Value Objects (Email).
// Ficam no pacote `vo` para evitar dependência circular com `user`.

var (
	// ErrInvalidEmail indica que o email informado não é válido.
	ErrInvalidEmail = errors.New("email inválido")

	// ErrInvalidID is returned when a string is not a valid UUID v7.
	ErrInvalidID = errors.New("invalid ID")

	// ErrInvalidPassword is returned when a password does not match the stored hash.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrPasswordTooShort is returned when password has less than 8 characters.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// ErrPasswordNoLetter is returned when password contains no letters.
	ErrPasswordNoLetter = errors.New("password must contain at least one letter")

	// ErrPasswordNoNumber is returned when password contains no digits.
	ErrPasswordNoNumber = errors.New("password must contain at least one number")

	// ErrPasswordNoSpecial is returned when password contains no special characters.
	ErrPasswordNoSpecial = errors.New("password must contain at least one special character")
)

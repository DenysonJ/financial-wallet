package vo

import (
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// DefaultBcryptCost is the default bcrypt hashing cost.
const DefaultBcryptCost = 12

// Password represents a hashed password.
type Password string

// NewPassword validates the plain-text password and returns a bcrypt hash.
// Uses the provided cost, or DefaultBcryptCost if cost <= 0.
func NewPassword(plain string, cost int) (Password, error) {
	if validateErr := ValidatePasswordStrength(plain); validateErr != nil {
		return "", validateErr
	}

	if cost <= 0 {
		cost = DefaultBcryptCost
	}

	hash, hashErr := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if hashErr != nil {
		return "", hashErr
	}

	return Password(hash), nil
}

// CheckPassword verifies a plain-text password against a bcrypt hash.
// Returns ErrInvalidPassword if the password does not match or the hash is malformed.
func CheckPassword(hash, plain string) error {
	if !isValidBcryptHash(hash) {
		return ErrInvalidPassword
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); compareErr != nil {
		return ErrInvalidPassword
	}
	return nil
}

// isValidBcryptHash checks that hash has a valid bcrypt format: $2a$XX$<53 chars>.
// Rejects malformed or high-cost hashes to prevent slow processing.
func isValidBcryptHash(hash string) bool {
	const maxBcryptCost = 16 // costs above this are impractically slow

	if len(hash) != 60 {
		return false
	}
	// Prefix: $2a$, $2b$, or $2y$
	if hash[0] != '$' || hash[1] != '2' ||
		(hash[2] != 'a' && hash[2] != 'b' && hash[2] != 'y') ||
		hash[3] != '$' || hash[6] != '$' {
		return false
	}
	// Cost: two-digit number between bcrypt.MinCost and maxBcryptCost
	cost := int(hash[4]-'0')*10 + int(hash[5]-'0')
	return cost >= bcrypt.MinCost && cost <= maxBcryptCost
}

// String returns the hash string.
func (p Password) String() string {
	return string(p)
}

// ValidatePasswordStrength checks that a password meets complexity requirements:
// - At least 8 characters
// - At least 1 letter
// - At least 1 number
// - At least 1 special character
func ValidatePasswordStrength(plain string) error {
	if len(plain) < 8 {
		return ErrPasswordTooShort
	}

	var hasLetter, hasNumber, hasSpecial bool
	for _, ch := range plain {
		switch {
		case unicode.IsLetter(ch):
			hasLetter = true
		case unicode.IsDigit(ch):
			hasNumber = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	if !hasLetter {
		return ErrPasswordNoLetter
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}

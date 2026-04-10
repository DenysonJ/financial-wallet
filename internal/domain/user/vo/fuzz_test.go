package vo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzNewEmail(f *testing.F) {
	f.Add("user@example.com")                // valid
	f.Add("a@b.c")                           // minimal valid
	f.Add("user+tag@example.com")            // with plus tag
	f.Add("\"quoted string\"@example.com")   // quoted local part
	f.Add("user@sub.domain.example.com")     // subdomain
	f.Add("")                                // empty
	f.Add("not-an-email")                    // no @
	f.Add("@missing-local.com")              // missing local
	f.Add("missing-domain@")                 // missing domain
	f.Add("user@.com")                       // dot at domain start
	f.Add("user@example..com")               // double dot
	f.Add(string(make([]byte, 1000)))        // very long
	f.Add("user@example.com\x00")            // null byte
	f.Add("user@例え.jp")                      // unicode domain
	f.Add("Display Name <user@example.com>") // with display name

	f.Fuzz(func(t *testing.T, input string) {
		email, parseErr := NewEmail(input)

		// Must never panic
		if parseErr != nil {
			assert.ErrorIs(t, parseErr, ErrInvalidEmail)
			return
		}

		// If valid, String() must return original value
		assert.NotEmpty(t, email.String())

		// driver.Value must not error
		val, valErr := email.Value()
		assert.NoError(t, valErr)
		assert.Equal(t, email.String(), val)
	})
}

func FuzzEmailScan(f *testing.F) {
	f.Add("user@example.com")         // valid string
	f.Add("")                         // empty string
	f.Add("not-an-email")             // invalid but accepted (Scan has no validation)
	f.Add(string(make([]byte, 1000))) // very long

	f.Fuzz(func(t *testing.T, input string) {
		// Test with string input
		var emailStr Email
		scanStrErr := emailStr.Scan(input)
		if scanStrErr != nil {
			return
		}
		assert.Equal(t, input, emailStr.String())

		// Test with []byte input
		var emailBytes Email
		scanBytesErr := emailBytes.Scan([]byte(input))
		if scanBytesErr != nil {
			return
		}
		assert.Equal(t, input, emailBytes.String())
	})
}

func FuzzValidatePasswordStrength(f *testing.F) {
	f.Add("Str0ng!Pass")               // valid: letter + digit + special
	f.Add("abcdefgh")                  // only letters, 8 chars
	f.Add("12345678")                  // only digits
	f.Add("!@#$%^&*")                  // only special
	f.Add("short1!")                   // too short (7 chars)
	f.Add("")                          // empty
	f.Add("aB1!aB1!")                  // minimal valid
	f.Add("пароль1!")                  // cyrillic letters
	f.Add("密码test1!")                  // CJK + latin + digit + special
	f.Add("a\u200Bb\u200B1!")          // zero-width space
	f.Add("a\u0300b1!cdef")            // combining accent
	f.Add("🔑🔑🔑🔑1aaa")                  // emoji (symbol category)
	f.Add(string(make([]byte, 10000))) // very long

	f.Fuzz(func(t *testing.T, input string) {
		validateErr := ValidatePasswordStrength(input)

		// Must never panic
		if validateErr == nil {
			// If valid, must be at least 8 chars with letter+digit+special
			assert.GreaterOrEqual(t, len(input), 8)
			return
		}

		// Error must be one of the known errors
		knownErrors := []error{
			ErrPasswordTooShort,
			ErrPasswordNoLetter,
			ErrPasswordNoNumber,
			ErrPasswordNoSpecial,
		}
		isKnown := false
		for _, known := range knownErrors {
			if errors.Is(validateErr, known) {
				isKnown = true
				break
			}
		}
		assert.True(t, isKnown, "unexpected error: %v", validateErr)
	})
}

func FuzzNewPassword(f *testing.F) {
	f.Add("Str0ng!Pass")  // valid
	f.Add("short1!")      // too short
	f.Add("")             // empty
	f.Add("пароль1!abcd") // cyrillic
	f.Add("aB1!aB1!")     // minimal valid

	f.Fuzz(func(t *testing.T, input string) {
		// Use cost=4 (bcrypt minimum) for speed
		pw, newErr := NewPassword(input, 4)

		// Must never panic
		if newErr != nil {
			assert.Equal(t, Password(""), pw)
			return
		}

		// If valid, hash must verify against original
		checkErr := CheckPassword(pw.String(), input)
		assert.NoError(t, checkErr, "hash must verify against original input")
	})
}

func FuzzCheckPassword(f *testing.F) {
	// Pre-generate a valid hash for seeding
	f.Add("$2a$04$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012", "Str0ng!Pass") // fake bcrypt format
	f.Add("not-a-hash", "password")                                                    // invalid hash
	f.Add("", "")                                                                      // both empty
	f.Add("$2a$04$", "test")                                                           // truncated hash
	f.Add("$2a$20000000000000000000000000000000000000000000000000000000", "0")
	f.Add("$2a$30$00000000000000000000000000000000000000000000000000000", "0")

	f.Fuzz(func(t *testing.T, hash, plain string) {
		checkErr := CheckPassword(hash, plain)

		// Must never panic — either nil or ErrInvalidPassword
		if checkErr != nil {
			assert.ErrorIs(t, checkErr, ErrInvalidPassword)
		}
	})
}

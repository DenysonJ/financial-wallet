package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestNewPassword_Success(t *testing.T) {
	pw, pwErr := NewPassword("Str0ng!Passw", 4) // low cost for fast tests
	assert.NoError(t, pwErr)
	assert.NotEmpty(t, pw.String())

	// Verify it's a valid bcrypt hash
	compareErr := bcrypt.CompareHashAndPassword([]byte(pw.String()), []byte("Str0ng!Passw"))
	assert.NoError(t, compareErr)
}

func TestNewPassword_TooShort(t *testing.T) {
	_, pwErr := NewPassword("Ab1!", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordTooShort)
}

func TestNewPassword_NoLetter(t *testing.T) {
	_, pwErr := NewPassword("123456789012!", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordNoLetter)
}

func TestNewPassword_NoNumber(t *testing.T) {
	_, pwErr := NewPassword("Abcdefghijkl!", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordNoNumber)
}

func TestNewPassword_NoSpecial(t *testing.T) {
	_, pwErr := NewPassword("Abcdefghijk1", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordNoSpecial)
}

func TestNewPassword_DefaultCost(t *testing.T) {
	// cost <= 0 should use DefaultBcryptCost — just verify it doesn't error
	pw, pwErr := NewPassword("Str0ng!Passw", 0)
	assert.NoError(t, pwErr)
	assert.NotEmpty(t, pw.String())
}

func TestCheckPassword_Correct(t *testing.T) {
	pw, _ := NewPassword("Str0ng!Passw", 4)
	checkErr := CheckPassword(pw.String(), "Str0ng!Passw")
	assert.NoError(t, checkErr)
}

func TestCheckPassword_Incorrect(t *testing.T) {
	pw, _ := NewPassword("Str0ng!Passw", 4)
	checkErr := CheckPassword(pw.String(), "WrongPassw1!")
	assert.ErrorIs(t, checkErr, ErrInvalidPassword)
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"GIVEN compliant password WHEN validating THEN returns nil", "Str0ng!Passw", nil},
		{"GIVEN below minimum length WHEN validating THEN returns ErrPasswordTooShort", "Ab1!", ErrPasswordTooShort},
		{"GIVEN no letter WHEN validating THEN returns ErrPasswordNoLetter", "123456789012!", ErrPasswordNoLetter},
		{"GIVEN no number WHEN validating THEN returns ErrPasswordNoNumber", "Abcdefghijkl!", ErrPasswordNoNumber},
		{"GIVEN no special WHEN validating THEN returns ErrPasswordNoSpecial", "Abcdefghijk1", ErrPasswordNoSpecial},
		{"GIVEN unicode letter WHEN validating THEN returns nil", "Ação12345678!", nil},
		{"GIVEN exactly minimum chars WHEN validating THEN returns nil", "Abcdefghij1!", nil},
		{"GIVEN one below minimum WHEN validating THEN returns ErrPasswordTooShort", "Abcdefghi1!", ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateErr := ValidatePasswordStrength(tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, validateErr, tt.wantErr)
			} else {
				assert.NoError(t, validateErr)
			}
		})
	}
}

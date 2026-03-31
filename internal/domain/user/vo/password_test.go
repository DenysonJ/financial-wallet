package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestNewPassword_Success(t *testing.T) {
	pw, pwErr := NewPassword("Str0ng!Pass", 4) // low cost for fast tests
	assert.NoError(t, pwErr)
	assert.NotEmpty(t, pw.String())

	// Verify it's a valid bcrypt hash
	compareErr := bcrypt.CompareHashAndPassword([]byte(pw.String()), []byte("Str0ng!Pass"))
	assert.NoError(t, compareErr)
}

func TestNewPassword_TooShort(t *testing.T) {
	_, pwErr := NewPassword("Ab1!", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordTooShort)
}

func TestNewPassword_NoLetter(t *testing.T) {
	_, pwErr := NewPassword("12345678!", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordNoLetter)
}

func TestNewPassword_NoNumber(t *testing.T) {
	_, pwErr := NewPassword("Abcdefgh!", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordNoNumber)
}

func TestNewPassword_NoSpecial(t *testing.T) {
	_, pwErr := NewPassword("Abcdefg1", 4)
	assert.ErrorIs(t, pwErr, ErrPasswordNoSpecial)
}

func TestNewPassword_DefaultCost(t *testing.T) {
	// cost <= 0 should use DefaultBcryptCost — just verify it doesn't error
	pw, pwErr := NewPassword("Str0ng!Pass", 0)
	assert.NoError(t, pwErr)
	assert.NotEmpty(t, pw.String())
}

func TestCheckPassword_Correct(t *testing.T) {
	pw, _ := NewPassword("Str0ng!Pass", 4)
	checkErr := CheckPassword(pw.String(), "Str0ng!Pass")
	assert.NoError(t, checkErr)
}

func TestCheckPassword_Incorrect(t *testing.T) {
	pw, _ := NewPassword("Str0ng!Pass", 4)
	checkErr := CheckPassword(pw.String(), "WrongPass1!")
	assert.ErrorIs(t, checkErr, ErrInvalidPassword)
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"valid", "Str0ng!Pass", nil},
		{"too short", "Ab1!", ErrPasswordTooShort},
		{"no letter", "12345678!", ErrPasswordNoLetter},
		{"no number", "Abcdefgh!", ErrPasswordNoNumber},
		{"no special", "Abcdefg1", ErrPasswordNoSpecial},
		{"unicode letter", "Ação1234!", nil},
		{"exactly 8 chars", "Abcde1!", ErrPasswordTooShort},
		{"exactly 8 valid", "Abcdef1!", nil},
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

package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzNewAccountType(f *testing.F) {
	f.Add("bank_account")              // valid
	f.Add("credit_card")               // valid
	f.Add("cash")                      // valid
	f.Add("")                          // empty
	f.Add("savings")                   // unknown type
	f.Add("BANK_ACCOUNT")              // uppercase
	f.Add("Bank_Account")              // mixed case
	f.Add(" cash")                     // leading space
	f.Add("cash ")                     // trailing space
	f.Add("bank_account\x00")          // null byte
	f.Add("bаnk_account")              // cyrillic 'а' lookalike
	f.Add(string(make([]byte, 10000))) // very long

	f.Fuzz(func(t *testing.T, input string) {
		at, typeErr := NewAccountType(input)

		// Must never panic
		if typeErr != nil {
			assert.ErrorIs(t, typeErr, ErrInvalidAccountType)
			assert.Equal(t, AccountType(""), at)
			return
		}

		// If valid, must be one of the known types
		validTypes := map[AccountType]bool{
			TypeBankAccount: true,
			TypeCreditCard:  true,
			TypeCash:        true,
		}
		assert.True(t, validTypes[at], "unexpected valid type: %s", at)

		// String() must match input
		assert.Equal(t, input, at.String())

		// driver.Value must round-trip
		val, valErr := at.Value()
		assert.NoError(t, valErr)
		assert.Equal(t, input, val)
	})
}

func FuzzAccountTypeScan(f *testing.F) {
	f.Add("bank_account")             // valid string
	f.Add("credit_card")              // valid string
	f.Add("")                         // empty
	f.Add("unknown")                  // invalid but accepted (Scan has no validation)
	f.Add(string(make([]byte, 1000))) // very long

	f.Fuzz(func(t *testing.T, input string) {
		// Test string path
		var atStr AccountType
		scanStrErr := atStr.Scan(input)
		if scanStrErr != nil {
			return
		}
		assert.Equal(t, AccountType(input), atStr)

		// Test []byte path
		var atBytes AccountType
		scanBytesErr := atBytes.Scan([]byte(input))
		if scanBytesErr != nil {
			return
		}
		assert.Equal(t, AccountType(input), atBytes)

		// Both paths must produce same result
		assert.Equal(t, atStr, atBytes)
	})
}

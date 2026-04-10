package vo

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func FuzzParseID(f *testing.F) {
	// Seed corpus
	f.Add(uuid.Must(uuid.NewV7()).String())           // valid UUID v7
	f.Add("550e8400-e29b-41d4-a716-446655440000")     // valid UUID v4
	f.Add("")                                         // empty
	f.Add("not-a-uuid")                               // invalid
	f.Add("550e8400-e29b-41d4")                       // partial
	f.Add("  550e8400-e29b-41d4-a716-446655440000  ") // with spaces
	f.Add("550e8400-e29b-41d4-a716-446655440000\x00") // null byte
	f.Add("XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX")     // right format, wrong chars
	f.Add(string(make([]byte, 10000)))                // very long string

	f.Fuzz(func(t *testing.T, input string) {
		id, parseErr := ParseID(input)

		// Must never panic — either valid or error
		if parseErr != nil {
			assert.Equal(t, ID(""), id, "on error, ID must be zero value")
			assert.ErrorIs(t, parseErr, ErrInvalidID)
			return
		}

		// If valid, must round-trip
		assert.Equal(t, input, id.String())

		// Re-parse must also succeed
		id2, reparseErr := ParseID(id.String())
		assert.NoError(t, reparseErr)
		assert.Equal(t, id, id2)
	})
}

func FuzzIDScan(f *testing.F) {
	f.Add(uuid.Must(uuid.NewV7()).String()) // valid UUID string
	f.Add("")                               // empty string
	f.Add("not-a-uuid")                     // invalid
	f.Add(string(make([]byte, 10000)))      // very long

	f.Fuzz(func(t *testing.T, input string) {
		var id ID
		scanErr := id.Scan(input)

		// Must never panic
		if scanErr != nil {
			assert.Equal(t, ID(""), id, "on error, ID must remain zero value")
			return
		}

		// If valid, must match input
		assert.Equal(t, ID(input), id)

		// driver.Value must round-trip
		val, valErr := id.Value()
		assert.NoError(t, valErr)
		assert.Equal(t, input, val)
	})
}

package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	id := NewID()

	// ID deve ser UUID v7 válido (36 caracteres com hífens)
	assert.Len(t, id.String(), 36)
	assert.NotEmpty(t, id.String())

	// Cada chamada gera ID único
	id2 := NewID()
	assert.NotEqual(t, id, id2)
}

func TestParseID_Valid(t *testing.T) {
	// Gera um ID válido para testar
	original := NewID()

	parsed, err := ParseID(original.String())

	require.NoError(t, err)
	assert.Equal(t, original, parsed)
}

func TestParseID_Invalid(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too short", "018e4a2c"},
		{"not a uuid", "not-a-valid-uuid-string"},
		{"random string", "invalid-id-format-that-is-not-uuid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseID(tc.input)
			assert.Error(t, err)
		})
	}
}

func TestID_ScanValue(t *testing.T) {
	original := NewID()

	// Test Value
	value, err := original.Value()
	require.NoError(t, err)
	assert.Equal(t, original.String(), value)

	// Test Scan from string
	var scanned ID
	err = scanned.Scan(original.String())
	require.NoError(t, err)
	assert.Equal(t, original, scanned)
}

func TestID_Scan_Error(t *testing.T) {
	var id ID

	err := id.Scan(nil)
	assert.Error(t, err)

	err = id.Scan(123)
	assert.Error(t, err)

	// []byte is not supported by the new UUID-based Scan
	err = id.Scan([]byte("018e4a2c-6b4d-7000-9410-abcdef123456"))
	assert.Error(t, err)
}

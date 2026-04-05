package vo

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "gera ID não vazio"},
		{name: "IDs consecutivos são diferentes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := NewID()
			assert.NotEmpty(t, id)
			assert.NotEqual(t, ID(""), id)

			// Valid UUID
			_, parseErr := uuid.Parse(id.String())
			assert.NoError(t, parseErr)
		})
	}

	// Uniqueness
	id1 := NewID()
	id2 := NewID()
	assert.NotEqual(t, id1, id2)
}

func TestParseID(t *testing.T) {
	validUUID := uuid.Must(uuid.NewV7()).String()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "UUID v7 válido", input: validUUID},
		{name: "UUID v4 válido", input: "550e8400-e29b-41d4-a716-446655440000"},
		{name: "string vazia", input: "", wantErr: ErrInvalidID},
		{name: "string inválida", input: "not-a-uuid", wantErr: ErrInvalidID},
		{name: "UUID parcial", input: "550e8400-e29b-41d4", wantErr: ErrInvalidID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, parseErr := ParseID(tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, parseErr, tt.wantErr)
				assert.Equal(t, ID(""), id)
			} else {
				assert.NoError(t, parseErr)
				assert.Equal(t, ID(tt.input), id)
			}
		})
	}
}

func TestID_String(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want string
	}{
		{name: "retorna string do ID", id: ID("abc-123"), want: "abc-123"},
		{name: "ID vazio", id: ID(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.id.String())
		})
	}
}

func TestID_Value(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want string
	}{
		{name: "retorna driver.Value", id: ID("abc-123"), want: "abc-123"},
		{name: "ID vazio", id: ID(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, valErr := tt.id.Value()
			assert.NoError(t, valErr)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestID_Scan(t *testing.T) {
	validUUID := uuid.Must(uuid.NewV7()).String()

	tests := []struct {
		name    string
		input   any
		want    ID
		wantErr bool
		errMsg  string
	}{
		{name: "string válida", input: validUUID, want: ID(validUUID)},
		{name: "nil retorna erro", input: nil, wantErr: true, errMsg: "ID cannot be empty"},
		{name: "tipo inválido (int)", input: 123, wantErr: true, errMsg: "invalid type for ID"},
		{name: "UUID inválido", input: "not-a-uuid", wantErr: true, errMsg: "invalid ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ID
			scanErr := id.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, scanErr)
				assert.Contains(t, scanErr.Error(), tt.errMsg)
			} else {
				assert.NoError(t, scanErr)
				assert.Equal(t, tt.want, id)
			}
		})
	}
}

package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCategoryType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CategoryType
		wantErr error
	}{
		{name: "GIVEN 'credit' WHEN NewCategoryType THEN returns TypeCredit", input: "credit", want: TypeCredit},
		{name: "GIVEN 'debit' WHEN NewCategoryType THEN returns TypeDebit", input: "debit", want: TypeDebit},
		{name: "GIVEN empty string WHEN NewCategoryType THEN returns ErrInvalidCategoryType", input: "", wantErr: ErrInvalidCategoryType},
		{name: "GIVEN unknown value WHEN NewCategoryType THEN returns ErrInvalidCategoryType", input: "transfer", wantErr: ErrInvalidCategoryType},
		{name: "GIVEN uppercase WHEN NewCategoryType THEN returns ErrInvalidCategoryType", input: "CREDIT", wantErr: ErrInvalidCategoryType},
		{name: "GIVEN leading whitespace WHEN NewCategoryType THEN returns ErrInvalidCategoryType", input: " debit", wantErr: ErrInvalidCategoryType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got, typeErr := NewCategoryType(tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, typeErr, tt.wantErr)
				assert.Equal(t, CategoryType(""), got)
				return
			}
			assert.NoError(t, typeErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCategoryType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CategoryType
	}{
		{name: "GIVEN valid type WHEN ParseCategoryType THEN returns the typed value", input: "credit", want: TypeCredit},
		{name: "GIVEN unknown value WHEN ParseCategoryType THEN accepts without validation (DB read path)", input: "unknown", want: CategoryType("unknown")},
		{name: "GIVEN empty string WHEN ParseCategoryType THEN returns empty CategoryType", input: "", want: CategoryType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.Equal(t, tt.want, ParseCategoryType(tt.input))
		})
	}
}

func TestCategoryType_String(t *testing.T) {
	tests := []struct {
		name string
		t    CategoryType
		want string
	}{
		{name: "GIVEN TypeCredit WHEN String THEN returns 'credit'", t: TypeCredit, want: "credit"},
		{name: "GIVEN TypeDebit WHEN String THEN returns 'debit'", t: TypeDebit, want: "debit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.Equal(t, tt.want, tt.t.String())
		})
	}
}

func TestCategoryType_Value(t *testing.T) {
	tests := []struct {
		name string
		t    CategoryType
		want string
	}{
		{name: "GIVEN TypeCredit WHEN Value THEN returns string 'credit'", t: TypeCredit, want: "credit"},
		{name: "GIVEN TypeDebit WHEN Value THEN returns string 'debit'", t: TypeDebit, want: "debit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			val, valErr := tt.t.Value()

			// Assert
			assert.NoError(t, valErr)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestCategoryType_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    CategoryType
		wantErr bool
		errMsg  string
	}{
		{name: "GIVEN valid string WHEN Scan THEN populates CategoryType", input: "credit", want: TypeCredit},
		{name: "GIVEN valid []byte WHEN Scan THEN populates CategoryType", input: []byte("debit"), want: TypeDebit},
		{name: "GIVEN nil WHEN Scan THEN returns 'cannot be nil' error", input: nil, wantErr: true, errMsg: "category type cannot be nil"},
		{name: "GIVEN int WHEN Scan THEN returns 'invalid type' error", input: 123, wantErr: true, errMsg: "invalid type for CategoryType"},
		{name: "GIVEN bool WHEN Scan THEN returns 'invalid type' error", input: true, wantErr: true, errMsg: "invalid type for CategoryType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var ct CategoryType

			// Act
			scanErr := ct.Scan(tt.input)

			// Assert
			if tt.wantErr {
				require.Error(t, scanErr)
				assert.Contains(t, scanErr.Error(), tt.errMsg)
				return
			}
			assert.NoError(t, scanErr)
			assert.Equal(t, tt.want, ct)
		})
	}
}

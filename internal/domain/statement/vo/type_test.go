package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatementType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StatementType
		wantErr error
	}{
		{name: "credit válido", input: "credit", want: TypeCredit},
		{name: "debit válido", input: "debit", want: TypeDebit},
		{name: "string vazia", input: "", wantErr: ErrInvalidStatementType},
		{name: "tipo desconhecido", input: "transfer", wantErr: ErrInvalidStatementType},
		{name: "uppercase", input: "CREDIT", wantErr: ErrInvalidStatementType},
		{name: "com espaço", input: " debit", wantErr: ErrInvalidStatementType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, typeErr := NewStatementType(tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, typeErr, tt.wantErr)
				assert.Equal(t, StatementType(""), got)
			} else {
				assert.NoError(t, typeErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseStatementType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  StatementType
	}{
		{name: "tipo válido", input: "credit", want: TypeCredit},
		{name: "tipo inválido aceito sem validação", input: "unknown", want: StatementType("unknown")},
		{name: "string vazia", input: "", want: StatementType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStatementType(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatementType_Opposite(t *testing.T) {
	tests := []struct {
		name string
		st   StatementType
		want StatementType
	}{
		{name: "credit → debit", st: TypeCredit, want: TypeDebit},
		{name: "debit → credit", st: TypeDebit, want: TypeCredit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.st.Opposite())
		})
	}
}

func TestStatementType_String(t *testing.T) {
	tests := []struct {
		name string
		st   StatementType
		want string
	}{
		{name: "credit", st: TypeCredit, want: "credit"},
		{name: "debit", st: TypeDebit, want: "debit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.st.String())
		})
	}
}

func TestStatementType_Value(t *testing.T) {
	tests := []struct {
		name string
		st   StatementType
		want string
	}{
		{name: "credit", st: TypeCredit, want: "credit"},
		{name: "debit", st: TypeDebit, want: "debit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, valErr := tt.st.Value()
			assert.NoError(t, valErr)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestStatementType_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    StatementType
		wantErr bool
		errMsg  string
	}{
		{name: "string válida", input: "credit", want: TypeCredit},
		{name: "[]byte válido", input: []byte("debit"), want: TypeDebit},
		{name: "nil retorna erro", input: nil, wantErr: true, errMsg: "statement type cannot be nil"},
		{name: "tipo inválido (int)", input: 123, wantErr: true, errMsg: "invalid type for StatementType"},
		{name: "tipo inválido (bool)", input: true, wantErr: true, errMsg: "invalid type for StatementType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st StatementType
			scanErr := st.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, scanErr)
				assert.Contains(t, scanErr.Error(), tt.errMsg)
			} else {
				assert.NoError(t, scanErr)
				assert.Equal(t, tt.want, st)
			}
		})
	}
}

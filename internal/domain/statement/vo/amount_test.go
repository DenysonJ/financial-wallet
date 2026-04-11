package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    Amount
		wantErr error
	}{
		{name: "valor positivo", input: 100, want: Amount(100)},
		{name: "1 centavo", input: 1, want: Amount(1)},
		{name: "valor grande", input: 99999999, want: Amount(99999999)},
		{name: "zero", input: 0, wantErr: ErrInvalidAmount},
		{name: "negativo", input: -50, wantErr: ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, amountErr := NewAmount(tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, amountErr, tt.wantErr)
				assert.Equal(t, Amount(0), got)
			} else {
				assert.NoError(t, amountErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  Amount
	}{
		{name: "valor positivo", input: 500, want: Amount(500)},
		{name: "zero aceito sem validação", input: 0, want: Amount(0)},
		{name: "negativo aceito sem validação", input: -10, want: Amount(-10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAmount(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAmount_Int64(t *testing.T) {
	a := Amount(1500)
	assert.Equal(t, int64(1500), a.Int64())
}

func TestAmount_Value(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		want int64
	}{
		{name: "valor normal", a: Amount(1000), want: 1000},
		{name: "1 centavo", a: Amount(1), want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, valErr := tt.a.Value()
			assert.NoError(t, valErr)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestAmount_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    Amount
		wantErr bool
		errMsg  string
	}{
		{name: "int64 válido", input: int64(2500), want: Amount(2500)},
		{name: "float64 válido", input: float64(3000), want: Amount(3000)},
		{name: "nil retorna erro", input: nil, wantErr: true, errMsg: "amount cannot be nil"},
		{name: "tipo inválido (string)", input: "100", wantErr: true, errMsg: "invalid type for Amount"},
		{name: "tipo inválido (bool)", input: true, wantErr: true, errMsg: "invalid type for Amount"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a Amount
			scanErr := a.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, scanErr)
				assert.Contains(t, scanErr.Error(), tt.errMsg)
			} else {
				assert.NoError(t, scanErr)
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

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
		{name: "given positive value when creating then succeeds", input: 100, want: Amount(100)},
		{name: "given 1 cent when creating then succeeds", input: 1, want: Amount(1)},
		{name: "given large value when creating then succeeds", input: 99999999, want: Amount(99999999)},
		{name: "given zero when creating then returns error", input: 0, wantErr: ErrInvalidAmount},
		{name: "given negative when creating then returns error", input: -50, wantErr: ErrInvalidAmount},
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
		{name: "given positive value when parsing then succeeds", input: 500, want: Amount(500)},
		{name: "given zero when parsing then accepts without validation", input: 0, want: Amount(0)},
		{name: "given negative when parsing then accepts without validation", input: -10, want: Amount(-10)},
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
		{name: "given normal value when getting driver value then returns int64", a: Amount(1000), want: 1000},
		{name: "given 1 cent when getting driver value then returns int64", a: Amount(1), want: 1},
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
		{name: "given int64 when scanning then succeeds", input: int64(2500), want: Amount(2500)},
		{name: "given float64 when scanning then succeeds", input: float64(3000), want: Amount(3000)},
		{name: "given nil when scanning then returns error", input: nil, wantErr: true, errMsg: "amount cannot be nil"},
		{name: "given string when scanning then returns error", input: "100", wantErr: true, errMsg: "invalid type for Amount"},
		{name: "given bool when scanning then returns error", input: true, wantErr: true, errMsg: "invalid type for Amount"},
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

package ofx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:  "given positive decimal when parsing then returns cents",
			input: "150.75",
			want:  15075,
		},
		{
			name:  "given negative decimal when parsing then returns negative cents",
			input: "-30.50",
			want:  -3050,
		},
		{
			name:  "given whole number when parsing then returns cents with two zeros",
			input: "100",
			want:  10000,
		},
		{
			name:  "given single decimal place when parsing then pads to cents",
			input: "-7.5",
			want:  -750,
		},
		{
			name:  "given three decimal places when parsing then rounds to cents",
			input: "10.456",
			want:  1046,
		},
		{
			name:  "given three decimal places rounding up when parsing then rounds correctly",
			input: "10.455",
			want:  1046,
		},
		{
			name:  "given zero when parsing then returns zero",
			input: "0",
			want:  0,
		},
		{
			name:  "given zero with decimals when parsing then returns zero",
			input: "0.00",
			want:  0,
		},
		{
			name:  "given trailing dot when parsing then returns whole cents",
			input: "50.",
			want:  5000,
		},
		{
			name:  "given positive sign when parsing then returns positive cents",
			input: "+25.00",
			want:  2500,
		},
		{
			name:  "given whitespace when parsing then trims and returns cents",
			input: "  100.50  ",
			want:  10050,
		},
		{
			name:    "given empty string when parsing then returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:    "given non-numeric when parsing then returns error",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "given only sign when parsing then returns error",
			input:   "-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, parseErr := ParseAmount(tt.input)

			if tt.wantErr {
				assert.Error(t, parseErr)
				assert.ErrorIs(t, parseErr, ErrInvalidAmount)
			} else {
				assert.NoError(t, parseErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

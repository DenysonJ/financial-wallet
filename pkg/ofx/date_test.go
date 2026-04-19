package ofx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "given YYYYMMDD when parsing then returns date at midnight UTC",
			input: "20260115",
			want:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "given YYYYMMDDHHMMSS when parsing then returns full datetime UTC",
			input: "20260115120000",
			want:  time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:  "given datetime with fractional seconds when parsing then ignores fraction",
			input: "20260115120000.000",
			want:  time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:  "given datetime with negative timezone when parsing then converts to UTC",
			input: "20260115120000[-3:BRT]",
			want:  time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC),
		},
		{
			name:  "given datetime with positive timezone when parsing then converts to UTC",
			input: "20260115120000[5:EST]",
			want:  time.Date(2026, 1, 15, 7, 0, 0, 0, time.UTC),
		},
		{
			name:  "given date only with timezone when parsing then converts to UTC",
			input: "20260115[-3:BRT]",
			want:  time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC),
		},
		{
			name:  "given datetime with fraction and timezone when parsing then handles both",
			input: "20260115143000.500[-3:BRT]",
			want:  time.Date(2026, 1, 15, 17, 30, 0, 0, time.UTC),
		},
		{
			name:  "given whitespace around date when parsing then trims and parses",
			input: "  20260301  ",
			want:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "given empty string when parsing then returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:    "given too short string when parsing then returns error",
			input:   "2026",
			wantErr: true,
		},
		{
			name:    "given non-numeric when parsing then returns error",
			input:   "abcdefgh",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, parseErr := ParseDate(tt.input)

			if tt.wantErr {
				assert.Error(t, parseErr)
				assert.ErrorIs(t, parseErr, ErrInvalidDate)
			} else {
				assert.NoError(t, parseErr)
				assert.True(t, tt.want.Equal(got), "expected %v, got %v", tt.want, got)
			}
		})
	}
}

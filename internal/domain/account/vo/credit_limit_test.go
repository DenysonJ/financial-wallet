package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreditLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		wantErr bool
	}{
		{name: "GIVEN a typical limit WHEN creating THEN accepts it (INV-03)", input: 500000},
		{name: "GIVEN one cent WHEN creating THEN accepts it (INV-03)", input: 1},
		{name: "GIVEN exactly the ceiling WHEN creating THEN accepts it (INV-03)", input: MaxCreditLimit},
		{name: "GIVEN zero WHEN creating THEN rejects it (RN-03)", input: 0, wantErr: true},
		{name: "GIVEN a negative value WHEN creating THEN rejects it (RN-03)", input: -1, wantErr: true},
		{name: "GIVEN one cent above the ceiling WHEN creating THEN rejects it (RN-03)", input: MaxCreditLimit + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, limitErr := NewCreditLimit(tt.input)

			if tt.wantErr {
				assert.ErrorIs(t, limitErr, ErrInvalidCreditLimit)
				assert.Equal(t, CreditLimit(0), limit)
				return
			}

			require.NoError(t, limitErr)
			assert.Equal(t, tt.input, limit.Int64())
		})
	}
}

func TestCreditLimit_ValueRoundTrip(t *testing.T) {
	limit, limitErr := NewCreditLimit(500000)
	require.NoError(t, limitErr)

	value, valueErr := limit.Value()
	require.NoError(t, valueErr)
	assert.Equal(t, int64(500000), value)
}

func TestCreditLimit_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    int64
		wantErr bool
	}{
		{name: "GIVEN an int64 from the DB WHEN scanning THEN keeps the value", input: int64(500000), want: 500000},
		{name: "GIVEN a float64 from the DB WHEN scanning THEN truncates to cents", input: float64(500000), want: 500000},
		{name: "GIVEN NULL WHEN scanning THEN fails (absence is a nil pointer, not a zero limit)", input: nil, wantErr: true},
		{name: "GIVEN an unsupported type WHEN scanning THEN fails", input: "500000", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var limit CreditLimit
			scanErr := limit.Scan(tt.input)

			if tt.wantErr {
				assert.Error(t, scanErr)
				return
			}

			require.NoError(t, scanErr)
			assert.Equal(t, tt.want, limit.Int64())
		})
	}
}

func TestParseCreditLimit_SkipsValidation(t *testing.T) {
	// Leituras do banco não revalidam: a CHECK da migration já garante a faixa.
	assert.Equal(t, int64(0), ParseCreditLimit(0).Int64())
}

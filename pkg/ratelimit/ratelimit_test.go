package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResult_Fields(t *testing.T) {
	tests := []struct {
		name          string
		result        Result
		wantAllow     bool
		wantLimit     int
		wantRemaining int
	}{
		{
			name:          "request allowed",
			result:        Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(1 * time.Minute)},
			wantAllow:     true,
			wantLimit:     100,
			wantRemaining: 99,
		},
		{
			name:          "request denied",
			result:        Result{Allowed: false, Limit: 100, Remaining: 0, ResetAt: time.Now().Add(30 * time.Second)},
			wantAllow:     false,
			wantLimit:     100,
			wantRemaining: 0,
		},
		{
			name:          "zero values",
			result:        Result{},
			wantAllow:     false,
			wantLimit:     0,
			wantRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantAllow, tt.result.Allowed)
			assert.Equal(t, tt.wantLimit, tt.result.Limit)
			assert.Equal(t, tt.wantRemaining, tt.result.Remaining)
			assert.False(t, tt.result.ResetAt.IsZero() && tt.result.Allowed)
		})
	}
}

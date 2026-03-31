package role

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListFilter_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    ListFilter
		expected ListFilter
	}{
		{
			name:     "valores zero são normalizados",
			input:    ListFilter{Page: 0, Limit: 0},
			expected: ListFilter{Page: 1, Limit: 20},
		},
		{
			name:     "valores válidos são mantidos",
			input:    ListFilter{Page: 2, Limit: 50},
			expected: ListFilter{Page: 2, Limit: 50},
		},
		{
			name:     "limit maior que 100 é normalizado",
			input:    ListFilter{Page: 1, Limit: 200},
			expected: ListFilter{Page: 1, Limit: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Normalize()
			assert.Equal(t, tt.expected.Page, tt.input.Page)
			assert.Equal(t, tt.expected.Limit, tt.input.Limit)
		})
	}
}

func TestListFilter_Offset(t *testing.T) {
	tests := []struct {
		name   string
		filter ListFilter
		want   int
	}{
		{
			name:   "página 1 tem offset 0",
			filter: ListFilter{Page: 1, Limit: 20},
			want:   0,
		},
		{
			name:   "página 2 com limit 20 tem offset 20",
			filter: ListFilter{Page: 2, Limit: 20},
			want:   20,
		},
		{
			name:   "página 3 com limit 10 tem offset 20",
			filter: ListFilter{Page: 3, Limit: 10},
			want:   20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.filter.Offset())
		})
	}
}

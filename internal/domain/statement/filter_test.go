package statement

import (
	"testing"
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func TestListFilter_Normalize(t *testing.T) {
	tests := []struct {
		name      string
		filter    ListFilter
		wantPage  int
		wantLimit int
	}{
		{
			name:      "given zero page and limit when normalizing then applies defaults",
			filter:    ListFilter{Page: 0, Limit: 0},
			wantPage:  1,
			wantLimit: 20,
		},
		{
			name:      "given negative values when normalizing then applies defaults",
			filter:    ListFilter{Page: -1, Limit: -5},
			wantPage:  1,
			wantLimit: 20,
		},
		{
			name:      "given limit above maximum when normalizing then caps at 100",
			filter:    ListFilter{Page: 2, Limit: 200},
			wantPage:  2,
			wantLimit: 100,
		},
		{
			name:      "given valid values when normalizing then keeps them",
			filter:    ListFilter{Page: 3, Limit: 50},
			wantPage:  3,
			wantLimit: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.Normalize()
			assert.Equal(t, tt.wantPage, tt.filter.Page)
			assert.Equal(t, tt.wantLimit, tt.filter.Limit)
		})
	}
}

func TestListFilter_Offset(t *testing.T) {
	tests := []struct {
		name  string
		page  int
		limit int
		want  int
	}{
		{name: "given page 1 when calculating offset then returns 0", page: 1, limit: 20, want: 0},
		{name: "given page 2 when calculating offset then returns 20", page: 2, limit: 20, want: 20},
		{name: "given page 3 limit 10 when calculating offset then returns 20", page: 3, limit: 10, want: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ListFilter{Page: tt.page, Limit: tt.limit}
			assert.Equal(t, tt.want, f.Offset())
		})
	}
}

func TestListFilter_WithOptionalFields(t *testing.T) {
	now := time.Now()
	accountID := pkgvo.NewID()

	f := ListFilter{
		AccountID: accountID,
		Type:      new(vo.TypeCredit),
		DateFrom:  &now,
		DateTo:    &now,
	}

	assert.Equal(t, accountID, f.AccountID)
	assert.NotNil(t, f.Type)
	assert.Equal(t, vo.TypeCredit, *f.Type)
	assert.NotNil(t, f.DateFrom)
	assert.NotNil(t, f.DateTo)
}

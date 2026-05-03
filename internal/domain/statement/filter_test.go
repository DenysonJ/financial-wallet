package statement

import (
	"testing"
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestListFilter_CategoryAndTagsFilter(t *testing.T) {
	accountID := pkgvo.NewID()
	categoryID := pkgvo.NewID()
	tag1 := pkgvo.NewID()
	tag2 := pkgvo.NewID()
	debit := vo.TypeDebit

	tests := []struct {
		name           string
		filter         ListFilter
		wantCategoryID *pkgvo.ID
		wantTagIDs     []pkgvo.ID
		wantHasType    bool
	}{
		{
			name:           "GIVEN zero-value filter WHEN inspected THEN no category/tag constraints",
			filter:         ListFilter{AccountID: accountID},
			wantCategoryID: nil,
			wantTagIDs:     nil,
		},
		{
			name:           "GIVEN CategoryID set WHEN inspected THEN pointer is propagated",
			filter:         ListFilter{AccountID: accountID, CategoryID: &categoryID},
			wantCategoryID: &categoryID,
			wantTagIDs:     nil,
		},
		{
			name:           "GIVEN TagIDs set WHEN inspected THEN slice is preserved in order",
			filter:         ListFilter{AccountID: accountID, TagIDs: []pkgvo.ID{tag1, tag2}},
			wantCategoryID: nil,
			wantTagIDs:     []pkgvo.ID{tag1, tag2},
		},
		{
			name: "GIVEN category + tags + type combined WHEN inspected THEN all three are present",
			filter: ListFilter{
				AccountID:  accountID,
				Type:       &debit,
				CategoryID: &categoryID,
				TagIDs:     []pkgvo.ID{tag1},
			},
			wantCategoryID: &categoryID,
			wantTagIDs:     []pkgvo.ID{tag1},
			wantHasType:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.Equal(t, accountID, tt.filter.AccountID)
			if tt.wantCategoryID == nil {
				assert.Nil(t, tt.filter.CategoryID)
			} else {
				require.NotNil(t, tt.filter.CategoryID)
				assert.Equal(t, *tt.wantCategoryID, *tt.filter.CategoryID)
			}
			assert.Equal(t, tt.wantTagIDs, tt.filter.TagIDs)
			if tt.wantHasType {
				assert.NotNil(t, tt.filter.Type)
			}
		})
	}
}

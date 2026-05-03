package category

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFilter(t *testing.T) {
	uid := pkgvo.NewID()
	credit := vo.TypeCredit

	tests := []struct {
		name      string
		filter    ListFilter
		wantScope Scope
		wantType  *vo.CategoryType
	}{
		{
			name:      "GIVEN UserID only WHEN building ListFilter THEN Scope defaults to ScopeAll and Type is nil",
			filter:    ListFilter{UserID: uid},
			wantScope: ScopeAll,
			wantType:  nil,
		},
		{
			name:      "GIVEN Type=credit WHEN building ListFilter THEN Type pointer is set",
			filter:    ListFilter{UserID: uid, Type: &credit},
			wantScope: ScopeAll,
			wantType:  &credit,
		},
		{
			name:      "GIVEN explicit Scope=System WHEN building ListFilter THEN Scope is preserved",
			filter:    ListFilter{UserID: uid, Scope: ScopeSystem},
			wantScope: ScopeSystem,
			wantType:  nil,
		},
		{
			name:      "GIVEN explicit Scope=User WHEN building ListFilter THEN Scope is preserved",
			filter:    ListFilter{UserID: uid, Scope: ScopeUser},
			wantScope: ScopeUser,
			wantType:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.Equal(t, uid, tt.filter.UserID)
			assert.Equal(t, tt.wantScope, tt.filter.Scope)
			if tt.wantType == nil {
				assert.Nil(t, tt.filter.Type)
			} else {
				require.NotNil(t, tt.filter.Type)
				assert.Equal(t, *tt.wantType, *tt.filter.Type)
			}
		})
	}
}

package tag

import (
	"testing"

	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func TestListFilter(t *testing.T) {
	uid := pkgvo.NewID()

	tests := []struct {
		name      string
		filter    ListFilter
		wantScope Scope
	}{
		{
			name:      "GIVEN UserID only WHEN building ListFilter THEN Scope defaults to ScopeAll",
			filter:    ListFilter{UserID: uid},
			wantScope: ScopeAll,
		},
		{
			name:      "GIVEN explicit Scope=System WHEN building ListFilter THEN Scope is preserved",
			filter:    ListFilter{UserID: uid, Scope: ScopeSystem},
			wantScope: ScopeSystem,
		},
		{
			name:      "GIVEN explicit Scope=User WHEN building ListFilter THEN Scope is preserved",
			filter:    ListFilter{UserID: uid, Scope: ScopeUser},
			wantScope: ScopeUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.Equal(t, uid, tt.filter.UserID)
			assert.Equal(t, tt.wantScope, tt.filter.Scope)
		})
	}
}

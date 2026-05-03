package category

import (
	"context"
	"testing"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	"github.com/DenysonJ/financial-wallet/internal/mocks/categoryuci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/dto"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListUseCase_Execute(t *testing.T) {
	uid := pkgvo.NewID()

	tests := []struct {
		name        string
		input       dto.ListInput
		matchFilter func(categorydomain.ListFilter) bool
		repoResult  []*categorydomain.Category
		skipRepo    bool
		wantErr     error
		wantLen     int
		wantScopes  []string // expected scope of each output item, in order
	}{
		{
			name:  "GIVEN no filters WHEN Execute THEN scope=all and type=nil are propagated",
			input: dto.ListInput{UserID: uid.String()},
			matchFilter: func(f categorydomain.ListFilter) bool {
				return f.UserID == uid && f.Scope == categorydomain.ScopeAll && f.Type == nil
			},
			repoResult: []*categorydomain.Category{
				categorydomain.NewSystemCategory("Salário", categoryvo.TypeCredit),
				categorydomain.NewCategory(uid, "Mercado", categoryvo.TypeDebit),
			},
			wantLen:    2,
			wantScopes: []string{"system", "user"},
		},
		{
			name:  "GIVEN type=credit WHEN Execute THEN filter type is propagated",
			input: dto.ListInput{UserID: uid.String(), Type: "credit"},
			matchFilter: func(f categorydomain.ListFilter) bool {
				return f.Type != nil && *f.Type == categoryvo.TypeCredit
			},
			repoResult: []*categorydomain.Category{
				categorydomain.NewSystemCategory("Salário", categoryvo.TypeCredit),
			},
			wantLen:    1,
			wantScopes: []string{"system"},
		},
		{
			name:  "GIVEN scope=user WHEN Execute THEN filter scope is ScopeUser",
			input: dto.ListInput{UserID: uid.String(), Scope: "user"},
			matchFilter: func(f categorydomain.ListFilter) bool {
				return f.Scope == categorydomain.ScopeUser
			},
			repoResult: []*categorydomain.Category{},
			wantLen:    0,
		},
		{
			name:  "GIVEN scope=system WHEN Execute THEN filter scope is ScopeSystem",
			input: dto.ListInput{UserID: uid.String(), Scope: "system"},
			matchFilter: func(f categorydomain.ListFilter) bool {
				return f.Scope == categorydomain.ScopeSystem
			},
			repoResult: []*categorydomain.Category{},
			wantLen:    0,
		},
		{
			name:     "GIVEN invalid type WHEN Execute THEN returns ErrInvalidCategoryType (no DB call)",
			input:    dto.ListInput{UserID: uid.String(), Type: "x"},
			skipRepo: true,
			wantErr:  categoryvo.ErrInvalidCategoryType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := categoryuci.NewMockRepository(t)
			if !tt.skipRepo {
				repo.EXPECT().
					List(mock.Anything, mock.MatchedBy(tt.matchFilter)).
					Return(tt.repoResult, nil)
			}
			uc := NewListUseCase(repo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				return
			}
			require.NoError(t, execErr)
			require.Len(t, out.Data, tt.wantLen)
			for i, scope := range tt.wantScopes {
				assert.Equal(t, scope, out.Data[i].Scope, "scope at index %d", i)
			}
		})
	}
}

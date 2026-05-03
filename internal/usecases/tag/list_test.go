package tag

import (
	"context"
	"testing"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/mocks/taguci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/dto"
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
		matchFilter func(tagdomain.ListFilter) bool
		repoResult  []*tagdomain.Tag
		wantLen     int
		wantScopes  []string
	}{
		{
			name:  "GIVEN no scope WHEN Execute THEN returns defaults + own",
			input: dto.ListInput{UserID: uid.String()},
			matchFilter: func(f tagdomain.ListFilter) bool {
				return f.UserID == uid && f.Scope == tagdomain.ScopeAll
			},
			repoResult: []*tagdomain.Tag{
				tagdomain.NewSystemTag("recorrente"),
				tagdomain.NewTag(uid, "viagem"),
			},
			wantLen:    2,
			wantScopes: []string{"system", "user"},
		},
		{
			name:  "GIVEN scope=user WHEN Execute THEN filter scope is ScopeUser",
			input: dto.ListInput{UserID: uid.String(), Scope: "user"},
			matchFilter: func(f tagdomain.ListFilter) bool {
				return f.Scope == tagdomain.ScopeUser
			},
			repoResult: []*tagdomain.Tag{},
			wantLen:    0,
		},
		{
			name:  "GIVEN scope=system WHEN Execute THEN filter scope is ScopeSystem",
			input: dto.ListInput{UserID: uid.String(), Scope: "system"},
			matchFilter: func(f tagdomain.ListFilter) bool {
				return f.Scope == tagdomain.ScopeSystem
			},
			repoResult: []*tagdomain.Tag{},
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := taguci.NewMockRepository(t)
			repo.EXPECT().
				List(mock.Anything, mock.MatchedBy(tt.matchFilter)).
				Return(tt.repoResult, nil)
			uc := NewListUseCase(repo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			require.NoError(t, execErr)
			require.Len(t, out.Data, tt.wantLen)
			for i, scope := range tt.wantScopes {
				assert.Equal(t, scope, out.Data[i].Scope, "scope at index %d", i)
			}
		})
	}
}

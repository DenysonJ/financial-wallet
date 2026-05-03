package category

import (
	"testing"
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCategory(t *testing.T) {
	userID := pkgvo.NewID()

	tests := []struct {
		name         string
		factory      func() *Category
		wantUserID   *pkgvo.ID
		wantName     string
		wantType     vo.CategoryType
		wantIsSystem bool
	}{
		{
			name:         "GIVEN owner+name+type WHEN NewCategory THEN builds user-scoped category",
			factory:      func() *Category { return NewCategory(userID, "Mercado", vo.TypeDebit) },
			wantUserID:   &userID,
			wantName:     "Mercado",
			wantType:     vo.TypeDebit,
			wantIsSystem: false,
		},
		{
			name:         "GIVEN system seed WHEN NewSystemCategory THEN builds default with user_id nil",
			factory:      func() *Category { return NewSystemCategory("Salário", vo.TypeCredit) },
			wantUserID:   nil,
			wantName:     "Salário",
			wantType:     vo.TypeCredit,
			wantIsSystem: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			c := tt.factory()

			// Assert
			require.NotNil(t, c)
			assert.NotEmpty(t, c.ID)
			assert.Equal(t, tt.wantName, c.Name)
			assert.Equal(t, tt.wantType, c.Type)
			assert.NotZero(t, c.CreatedAt)
			assert.NotZero(t, c.UpdatedAt)
			assert.Equal(t, tt.wantIsSystem, c.IsSystem())
			if tt.wantUserID == nil {
				assert.Nil(t, c.UserID)
			} else {
				require.NotNil(t, c.UserID)
				assert.Equal(t, *tt.wantUserID, *c.UserID)
			}
		})
	}
}

func TestCategory_Rename(t *testing.T) {
	tests := []struct {
		name     string
		newName  string
		wantName string
	}{
		{
			name:     "GIVEN owned category WHEN Rename to non-empty THEN persists new name and preserves type",
			newName:  "Supermercado",
			wantName: "Supermercado",
		},
		{
			name:     "GIVEN owned category WHEN Rename to same name THEN no observable change beyond UpdatedAt",
			newName:  "Mercado",
			wantName: "Mercado",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			c := NewCategory(pkgvo.NewID(), "Mercado", vo.TypeDebit)
			oldUpdatedAt := c.UpdatedAt
			originalType := c.Type
			time.Sleep(time.Microsecond) // ensure UpdatedAt advances

			// Act
			renameErr := c.Rename(tt.newName)

			// Assert
			require.NoError(t, renameErr)
			assert.Equal(t, tt.wantName, c.Name)
			assert.Equal(t, originalType, c.Type, "Rename must NEVER change Type")
			assert.GreaterOrEqual(t, c.UpdatedAt.UnixNano(), oldUpdatedAt.UnixNano())
		})
	}
}

func TestCategory_IsSystem(t *testing.T) {
	tests := []struct {
		name    string
		factory func() *Category
		want    bool
	}{
		{
			name:    "GIVEN system default WHEN IsSystem THEN true",
			factory: func() *Category { return NewSystemCategory("Estorno", vo.TypeCredit) },
			want:    true,
		},
		{
			name:    "GIVEN user-owned category WHEN IsSystem THEN false",
			factory: func() *Category { return NewCategory(pkgvo.NewID(), "Mercado", vo.TypeDebit) },
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.Equal(t, tt.want, tt.factory().IsSystem())
		})
	}
}

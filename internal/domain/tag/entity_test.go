package tag

import (
	"testing"
	"time"

	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTag(t *testing.T) {
	userID := pkgvo.NewID()

	tests := []struct {
		name         string
		factory      func() *Tag
		wantUserID   *pkgvo.ID
		wantName     string
		wantIsSystem bool
	}{
		{
			name:         "GIVEN owner+name WHEN NewTag THEN builds user-scoped tag",
			factory:      func() *Tag { return NewTag(userID, "viagem-2026") },
			wantUserID:   &userID,
			wantName:     "viagem-2026",
			wantIsSystem: false,
		},
		{
			name:         "GIVEN system seed WHEN NewSystemTag THEN builds default with user_id nil",
			factory:      func() *Tag { return NewSystemTag("recorrente") },
			wantUserID:   nil,
			wantName:     "recorrente",
			wantIsSystem: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			tag := tt.factory()

			// Assert
			require.NotNil(t, tag)
			assert.NotEmpty(t, tag.ID)
			assert.Equal(t, tt.wantName, tag.Name)
			assert.NotZero(t, tag.CreatedAt)
			assert.NotZero(t, tag.UpdatedAt)
			assert.Equal(t, tt.wantIsSystem, tag.IsSystem())
			if tt.wantUserID == nil {
				assert.Nil(t, tag.UserID)
			} else {
				require.NotNil(t, tag.UserID)
				assert.Equal(t, *tt.wantUserID, *tag.UserID)
			}
		})
	}
}

func TestTag_Rename(t *testing.T) {
	tests := []struct {
		name     string
		newName  string
		wantName string
	}{
		{
			name:     "GIVEN owned tag WHEN Rename to non-empty THEN persists new name",
			newName:  "new",
			wantName: "new",
		},
		{
			name:     "GIVEN owned tag WHEN Rename to same name THEN no observable change beyond UpdatedAt",
			newName:  "old",
			wantName: "old",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tag := NewTag(pkgvo.NewID(), "old")
			oldUpdatedAt := tag.UpdatedAt
			time.Sleep(time.Microsecond) // ensure UpdatedAt advances

			// Act
			renameErr := tag.Rename(tt.newName)

			// Assert
			require.NoError(t, renameErr)
			assert.Equal(t, tt.wantName, tag.Name)
			assert.GreaterOrEqual(t, tag.UpdatedAt.UnixNano(), oldUpdatedAt.UnixNano())
		})
	}
}

func TestTag_IsSystem(t *testing.T) {
	tests := []struct {
		name    string
		factory func() *Tag
		want    bool
	}{
		{
			name:    "GIVEN system default WHEN IsSystem THEN true",
			factory: func() *Tag { return NewSystemTag("recorrente") },
			want:    true,
		},
		{
			name:    "GIVEN user-owned tag WHEN IsSystem THEN false",
			factory: func() *Tag { return NewTag(pkgvo.NewID(), "viagem") },
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

func TestMaxTagsPerStatement(t *testing.T) {
	t.Run("GIVEN spec REQ-8/REQ-10 WHEN reading MaxTagsPerStatement THEN limit equals 10", func(t *testing.T) {
		// Act + Assert
		assert.Equal(t, 10, MaxTagsPerStatement)
	})
}

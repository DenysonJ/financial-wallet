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

func TestUpdateUseCase_Execute(t *testing.T) {
	owner := pkgvo.NewID()
	owned := tagdomain.NewTag(owner, "old")
	systemDefault := tagdomain.NewSystemTag("recorrente")
	crossUser := tagdomain.NewTag(pkgvo.NewID(), "stranger")

	tests := []struct {
		name        string
		input       dto.UpdateInput
		findResult  *tagdomain.Tag
		findErr     error
		updateErr   error
		skipFind    bool
		skipUpdate  bool
		wantErr     error
		wantName    string
		wantSuccess bool
	}{
		{
			name:        "GIVEN owned tag WHEN rename THEN persists new name",
			input:       dto.UpdateInput{UserID: owner.String(), ID: owned.ID.String(), Name: "new"},
			findResult:  owned,
			wantSuccess: true,
			wantName:    "new",
		},
		{
			name:       "GIVEN system default WHEN update THEN returns ErrTagReadOnly",
			input:      dto.UpdateInput{UserID: owner.String(), ID: systemDefault.ID.String(), Name: "x"},
			findResult: systemDefault,
			skipUpdate: true,
			wantErr:    tagdomain.ErrTagReadOnly,
		},
		{
			name:       "GIVEN cross-user tag WHEN update THEN returns ErrTagNotFound",
			input:      dto.UpdateInput{UserID: owner.String(), ID: crossUser.ID.String(), Name: "y"},
			findResult: crossUser,
			skipUpdate: true,
			wantErr:    tagdomain.ErrTagNotFound,
		},
		{
			name:       "GIVEN whitespace-only name WHEN update THEN returns ErrTagInvalidName (no DB call)",
			input:      dto.UpdateInput{UserID: owner.String(), ID: pkgvo.NewID().String(), Name: "  "},
			skipFind:   true,
			skipUpdate: true,
			wantErr:    tagdomain.ErrTagInvalidName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := taguci.NewMockRepository(t)
			if !tt.skipFind {
				idVO, _ := pkgvo.ParseID(tt.input.ID)
				repo.EXPECT().FindByID(mock.Anything, idVO).Return(tt.findResult, tt.findErr)
			}
			if !tt.skipUpdate {
				repo.EXPECT().
					Update(mock.Anything, mock.AnythingOfType("*tag.Tag")).
					Return(tt.updateErr)
			}
			uc := NewUpdateUseCase(repo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
			if tt.wantSuccess {
				assert.Equal(t, tt.wantName, out.Name)
			}
		})
	}
}

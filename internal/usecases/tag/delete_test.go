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

func TestDeleteUseCase_Execute(t *testing.T) {
	owner := pkgvo.NewID()
	owned := tagdomain.NewTag(owner, "viagem")
	systemDefault := tagdomain.NewSystemTag("recorrente")
	crossUser := tagdomain.NewTag(pkgvo.NewID(), "stranger")
	missingID := pkgvo.NewID()

	tests := []struct {
		name       string
		input      dto.DeleteInput
		findResult *tagdomain.Tag
		findErr    error
		deleteErr  error
		skipDelete bool
		wantErr    error
	}{
		{
			name:       "GIVEN owned tag WHEN delete THEN succeeds (CASCADE handles statement_tags)",
			input:      dto.DeleteInput{UserID: owner.String(), ID: owned.ID.String()},
			findResult: owned,
		},
		{
			name:       "GIVEN system default WHEN delete THEN returns ErrTagReadOnly",
			input:      dto.DeleteInput{UserID: owner.String(), ID: systemDefault.ID.String()},
			findResult: systemDefault,
			skipDelete: true,
			wantErr:    tagdomain.ErrTagReadOnly,
		},
		{
			name:       "GIVEN cross-user tag WHEN delete THEN returns ErrTagNotFound",
			input:      dto.DeleteInput{UserID: owner.String(), ID: crossUser.ID.String()},
			findResult: crossUser,
			skipDelete: true,
			wantErr:    tagdomain.ErrTagNotFound,
		},
		{
			name:       "GIVEN nonexistent tag WHEN delete THEN returns ErrTagNotFound",
			input:      dto.DeleteInput{UserID: owner.String(), ID: missingID.String()},
			findErr:    tagdomain.ErrTagNotFound,
			skipDelete: true,
			wantErr:    tagdomain.ErrTagNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := taguci.NewMockRepository(t)
			idVO, _ := pkgvo.ParseID(tt.input.ID)
			repo.EXPECT().FindByID(mock.Anything, idVO).Return(tt.findResult, tt.findErr)
			if !tt.skipDelete {
				repo.EXPECT().Delete(mock.Anything, idVO).Return(tt.deleteErr)
			}
			uc := NewDeleteUseCase(repo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
			assert.Equal(t, tt.input.ID, out.ID)
			assert.NotEmpty(t, out.DeletedAt)
		})
	}
}

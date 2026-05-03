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

func TestDeleteUseCase_Execute(t *testing.T) {
	owner := pkgvo.NewID()
	owned := categorydomain.NewCategory(owner, "Mercado", categoryvo.TypeDebit)
	systemDefault := categorydomain.NewSystemCategory("Estorno", categoryvo.TypeCredit)
	crossUser := categorydomain.NewCategory(pkgvo.NewID(), "OtherCat", categoryvo.TypeDebit)

	tests := []struct {
		name       string
		input      dto.DeleteInput
		findResult *categorydomain.Category
		count      int
		countErr   error
		deleteErr  error
		skipCount  bool
		skipDelete bool
		wantErr    error
	}{
		{
			name:       "GIVEN owned unused category WHEN delete THEN succeeds",
			input:      dto.DeleteInput{UserID: owner.String(), ID: owned.ID.String()},
			findResult: owned,
			count:      0,
		},
		{
			name:       "GIVEN owned in-use category WHEN delete THEN returns ErrCategoryInUse (no DELETE)",
			input:      dto.DeleteInput{UserID: owner.String(), ID: owned.ID.String()},
			findResult: owned,
			count:      7,
			skipDelete: true,
			wantErr:    categorydomain.ErrCategoryInUse,
		},
		{
			name:       "GIVEN system default WHEN delete THEN returns ErrCategoryReadOnly",
			input:      dto.DeleteInput{UserID: owner.String(), ID: systemDefault.ID.String()},
			findResult: systemDefault,
			skipCount:  true,
			skipDelete: true,
			wantErr:    categorydomain.ErrCategoryReadOnly,
		},
		{
			name:       "GIVEN cross-user category WHEN delete THEN returns ErrCategoryNotFound",
			input:      dto.DeleteInput{UserID: owner.String(), ID: crossUser.ID.String()},
			findResult: crossUser,
			skipCount:  true,
			skipDelete: true,
			wantErr:    categorydomain.ErrCategoryNotFound,
		},
		{
			name:       "GIVEN FK race (count=0 but DELETE fails) WHEN delete THEN propagates ErrCategoryInUse",
			input:      dto.DeleteInput{UserID: owner.String(), ID: owned.ID.String()},
			findResult: owned,
			count:      0,
			deleteErr:  categorydomain.ErrCategoryInUse,
			wantErr:    categorydomain.ErrCategoryInUse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := categoryuci.NewMockRepository(t)
			idVO, _ := pkgvo.ParseID(tt.input.ID)
			repo.EXPECT().FindByID(mock.Anything, idVO).Return(tt.findResult, nil)
			if !tt.skipCount {
				repo.EXPECT().CountStatementsUsing(mock.Anything, idVO).Return(tt.count, tt.countErr)
			}
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

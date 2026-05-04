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

func TestUpdateUseCase_Execute(t *testing.T) {
	owner := pkgvo.NewID()
	otherUser := pkgvo.NewID()
	owned := categorydomain.NewCategory(owner, "Mercado", categoryvo.TypeDebit)
	systemDefault := categorydomain.NewSystemCategory("Estorno", categoryvo.TypeCredit)
	crossUser := categorydomain.NewCategory(otherUser, "OtherCat", categoryvo.TypeDebit)

	tests := []struct {
		name        string
		input       dto.UpdateInput
		findResult  *categorydomain.Category
		findErr     error
		updateErr   error
		skipFind    bool
		skipUpdate  bool
		wantErr     error
		wantName    string
		wantType    string
		wantSuccess bool
	}{
		{
			name:        "GIVEN owned category WHEN rename THEN persists new name and preserves type",
			input:       dto.UpdateInput{UserID: owner.String(), ID: owned.ID.String(), Name: "Supermercado"},
			findResult:  owned,
			wantSuccess: true,
			wantName:    "Supermercado",
			wantType:    "debit",
		},
		{
			name:       "GIVEN system default WHEN update THEN returns ErrCategoryReadOnly (no UPDATE)",
			input:      dto.UpdateInput{UserID: owner.String(), ID: systemDefault.ID.String(), Name: "Estorninho"},
			findResult: systemDefault,
			skipUpdate: true,
			wantErr:    categorydomain.ErrCategoryReadOnly,
		},
		{
			name:       "GIVEN cross-user category WHEN update THEN returns ErrCategoryNotFound (no oracle)",
			input:      dto.UpdateInput{UserID: owner.String(), ID: crossUser.ID.String(), Name: "X"},
			findResult: crossUser,
			skipUpdate: true,
			wantErr:    categorydomain.ErrCategoryNotFound,
		},
		{
			name:       "GIVEN nonexistent ID WHEN update THEN returns ErrCategoryNotFound",
			input:      dto.UpdateInput{UserID: owner.String(), ID: pkgvo.NewID().String(), Name: "Novo"},
			findErr:    categorydomain.ErrCategoryNotFound,
			skipUpdate: true,
			wantErr:    categorydomain.ErrCategoryNotFound,
		},
		{
			name:       "GIVEN whitespace-only name WHEN update THEN returns ErrCategoryInvalidName (no DB call)",
			input:      dto.UpdateInput{UserID: owner.String(), ID: pkgvo.NewID().String(), Name: "  "},
			skipFind:   true,
			skipUpdate: true,
			wantErr:    categorydomain.ErrCategoryInvalidName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := categoryuci.NewMockRepository(t)
			if !tt.skipFind {
				idVO, _ := pkgvo.ParseID(tt.input.ID)
				repo.EXPECT().FindByID(mock.Anything, idVO).Return(tt.findResult, tt.findErr)
			}
			if !tt.skipUpdate {
				repo.EXPECT().
					Update(mock.Anything, mock.AnythingOfType("*category.Category")).
					Return(tt.updateErr)
			}
			uc := NewUpdateUseCase(repo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.Nil(t, out)
				assert.ErrorIs(t, execErr, tt.wantErr)
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
			if tt.wantSuccess {
				assert.Equal(t, tt.wantName, out.Name)
				assert.Equal(t, tt.wantType, out.Type, "type must NEVER change in rename")
			}
		})
	}
}

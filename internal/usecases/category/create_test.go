package category

import (
	"context"
	"errors"
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

func TestCreateUseCase_Execute(t *testing.T) {
	uid := pkgvo.NewID()

	tests := []struct {
		name        string
		input       dto.CreateInput
		repoErr     error  // returned by repo.Create when it is called
		skipRepo    bool   // when true, repo.Create must NOT be called
		wantErr     error  // expected sentinel (ErrorIs)
		wantErrSub  string // optional substring of the error message
		wantName    string
		wantType    string
		wantScope   string
		wantSuccess bool
	}{
		{
			name:        "GIVEN valid input WHEN Execute THEN returns CategoryOutput with scope=user",
			input:       dto.CreateInput{UserID: uid.String(), Name: "Mercado", Type: "debit"},
			wantSuccess: true,
			wantName:    "Mercado",
			wantType:    "debit",
			wantScope:   "user",
		},
		{
			name:    "GIVEN duplicate WHEN Execute THEN returns ErrCategoryDuplicate",
			input:   dto.CreateInput{UserID: uid.String(), Name: "Mercado", Type: "debit"},
			repoErr: categorydomain.ErrCategoryDuplicate,
			wantErr: categorydomain.ErrCategoryDuplicate,
		},
		{
			name:     "GIVEN invalid type WHEN Execute THEN returns ErrInvalidCategoryType (no DB call)",
			input:    dto.CreateInput{UserID: uid.String(), Name: "X", Type: "transfer"},
			skipRepo: true,
			wantErr:  categoryvo.ErrInvalidCategoryType,
		},
		{
			name:     "GIVEN whitespace-only name WHEN Execute THEN returns ErrCategoryInvalidName",
			input:    dto.CreateInput{UserID: uid.String(), Name: "   ", Type: "debit"},
			skipRepo: true,
			wantErr:  categorydomain.ErrCategoryInvalidName,
		},
		{
			name:     "GIVEN invalid user_id WHEN Execute THEN returns ErrInvalidID",
			input:    dto.CreateInput{UserID: "not-a-uuid", Name: "X", Type: "debit"},
			skipRepo: true,
			wantErr:  pkgvo.ErrInvalidID,
		},
		{
			name:       "GIVEN repo infrastructure error WHEN Execute THEN propagates error",
			input:      dto.CreateInput{UserID: uid.String(), Name: "X", Type: "debit"},
			repoErr:    errors.New("connection refused"),
			wantErrSub: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := categoryuci.NewMockRepository(t)
			if !tt.skipRepo {
				repo.EXPECT().
					Create(mock.Anything, mock.AnythingOfType("*category.Category")).
					Return(tt.repoErr)
			}
			uc := NewCreateUseCase(repo)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			switch {
			case tt.wantSuccess:
				require.NoError(t, execErr)
				require.NotNil(t, out)
				assert.Equal(t, tt.wantName, out.Name)
				assert.Equal(t, tt.wantType, out.Type)
				assert.Equal(t, tt.wantScope, out.Scope)
				assert.NotEmpty(t, out.ID)
			case tt.wantErr != nil:
				assert.Nil(t, out)
				assert.ErrorIs(t, execErr, tt.wantErr)
			case tt.wantErrSub != "":
				assert.Nil(t, out)
				require.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.wantErrSub)
			}
		})
	}
}

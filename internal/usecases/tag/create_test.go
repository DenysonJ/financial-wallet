package tag

import (
	"context"
	"errors"
	"testing"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/mocks/taguci"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/dto"
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
		repoErr     error
		skipRepo    bool
		wantErr     error
		wantErrSub  string
		wantName    string
		wantScope   string
		wantSuccess bool
	}{
		{
			name:        "GIVEN valid input WHEN Execute THEN returns TagOutput with scope=user",
			input:       dto.CreateInput{UserID: uid.String(), Name: "viagem-2026"},
			wantSuccess: true,
			wantName:    "viagem-2026",
			wantScope:   "user",
		},
		{
			name:    "GIVEN duplicate WHEN Execute THEN returns ErrTagDuplicate",
			input:   dto.CreateInput{UserID: uid.String(), Name: "viagem"},
			repoErr: tagdomain.ErrTagDuplicate,
			wantErr: tagdomain.ErrTagDuplicate,
		},
		{
			name:     "GIVEN whitespace-only name WHEN Execute THEN returns ErrTagInvalidName (no DB call)",
			input:    dto.CreateInput{UserID: uid.String(), Name: "  "},
			skipRepo: true,
			wantErr:  tagdomain.ErrTagInvalidName,
		},
		{
			name:     "GIVEN invalid user_id WHEN Execute THEN returns ErrInvalidID",
			input:    dto.CreateInput{UserID: "not-a-uuid", Name: "x"},
			skipRepo: true,
			wantErr:  pkgvo.ErrInvalidID,
		},
		{
			name:       "GIVEN repo infrastructure error WHEN Execute THEN propagates error",
			input:      dto.CreateInput{UserID: uid.String(), Name: "x"},
			repoErr:    errors.New("boom"),
			wantErrSub: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := taguci.NewMockRepository(t)
			if !tt.skipRepo {
				repo.EXPECT().
					Create(mock.Anything, mock.AnythingOfType("*tag.Tag")).
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

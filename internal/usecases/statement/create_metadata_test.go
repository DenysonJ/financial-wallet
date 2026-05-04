package statement

import (
	"context"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// dedupTagIDs
// =============================================================================

func TestDedupTagIDs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "GIVEN empty input WHEN dedup THEN returns nil", in: nil, want: nil},
		{name: "GIVEN single id WHEN dedup THEN returns same id", in: []string{"a"}, want: []string{"a"}},
		{name: "GIVEN all unique WHEN dedup THEN preserves order", in: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "GIVEN duplicates WHEN dedup THEN drops duplicates preserving first-seen order", in: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "GIVEN all same WHEN dedup THEN returns single element", in: []string{"a", "a", "a"}, want: []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := dedupTagIDs(tt.in)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// CreateUseCase with category + tags
// =============================================================================

func newActiveAccount(t *testing.T, ownerID, accountID vo.ID) *accountdomain.Account {
	t.Helper()
	now := time.Now()
	return &accountdomain.Account{
		ID:        accountID,
		UserID:    ownerID,
		Name:      "Nubank",
		Type:      accountvo.TypeBankAccount,
		Active:    true,
		Balance:   10000,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestCreateUseCase_Execute_WithCategoryAndTags(t *testing.T) {
	ownerID := vo.NewID()
	accountID := vo.NewID()
	categoryID := vo.NewID()
	tag1 := vo.NewID()
	tag2 := vo.NewID()

	tooMany := make([]string, 0, 11)
	for i := 0; i < 11; i++ {
		tooMany = append(tooMany, vo.NewID().String())
	}

	tests := []struct {
		name             string
		input            dto.CreateInput
		categoryFix      *categorydomain.Category
		findCategoryErr  error
		visibleTags      []*tagdomain.Tag
		skipFindCategory bool
		skipFindManyVis  bool
		skipCreate       bool
		wantErr          error
		wantCategoryName string
		wantCategoryType string
		wantTagsCount    int
		wantNullCategory bool
		wantSuccess      bool
	}{
		{
			name: "GIVEN matching category + visible tags (with one duplicate) WHEN create THEN persists with metadata",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000, Description: "Compras do mês",
				CategoryID: ptrStr(categoryID.String()),
				TagIDs:     []string{tag1.String(), tag2.String(), tag1.String()},
			},
			categoryFix: &categorydomain.Category{
				ID: categoryID, UserID: &ownerID, Name: "Mercado", Type: categoryvo.TypeDebit,
			},
			visibleTags: []*tagdomain.Tag{
				{ID: tag1, UserID: &ownerID, Name: "viagem"},
				{ID: tag2, Name: "recorrente"},
			},
			wantSuccess:      true,
			wantCategoryName: "Mercado",
			wantCategoryType: "debit",
			wantTagsCount:    2,
		},
		{
			name:             "GIVEN no category and no tags WHEN create THEN emits null category and empty tags",
			input:            dto.CreateInput{Type: "debit", Amount: 2000},
			skipFindCategory: true,
			skipFindManyVis:  true,
			wantSuccess:      true,
			wantNullCategory: true,
			wantTagsCount:    0,
		},
		{
			name: "GIVEN category type mismatch WHEN create THEN returns ErrCategoryTypeMismatch",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000,
				CategoryID: ptrStr(categoryID.String()),
			},
			categoryFix: &categorydomain.Category{
				ID: categoryID, UserID: &ownerID, Name: "Salário", Type: categoryvo.TypeCredit,
			},
			skipFindManyVis: true,
			skipCreate:      true,
			wantErr:         categorydomain.ErrCategoryTypeMismatch,
		},
		{
			name: "GIVEN cross-user category WHEN create THEN returns ErrCategoryNotVisible (422 not 404)",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000,
				CategoryID: ptrStr(categoryID.String()),
			},
			findCategoryErr: categorydomain.ErrCategoryNotVisible,
			skipFindManyVis: true,
			skipCreate:      true,
			wantErr:         categorydomain.ErrCategoryNotVisible,
		},
		{
			name: "GIVEN invisible tag in array WHEN create THEN returns ErrTagNotVisible",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000,
				TagIDs: []string{tag1.String(), tag2.String()},
			},
			skipFindCategory: true,
			visibleTags:      []*tagdomain.Tag{{ID: tag1, UserID: &ownerID, Name: "viagem"}},
			skipCreate:       true,
			wantErr:          tagdomain.ErrTagNotVisible,
		},
		{
			name: "GIVEN >10 unique tag IDs WHEN create THEN returns ErrTagLimitExceeded (no DB call to find tags)",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000,
				TagIDs: tooMany,
			},
			skipFindCategory: true,
			skipFindManyVis:  true,
			skipCreate:       true,
			wantErr:          tagdomain.ErrTagLimitExceeded,
		},
		{
			name: "GIVEN invalid category UUID WHEN create THEN returns ErrInvalidID",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000,
				CategoryID: ptrStr("not-a-uuid"),
			},
			skipFindCategory: true,
			skipFindManyVis:  true,
			skipCreate:       true,
			wantErr:          vo.ErrInvalidID,
		},
		{
			name: "GIVEN empty CategoryID string WHEN create THEN treats as no category",
			input: dto.CreateInput{
				Type: "debit", Amount: 2000,
				CategoryID: ptrStr(""),
			},
			skipFindCategory: true,
			skipFindManyVis:  true,
			wantSuccess:      true,
			wantNullCategory: true,
			wantTagsCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			stmtRepo := &mockRepository{}
			accRepo := &mockAccountRepository{}
			catReader := &mockCategoryReader{}
			tagReader := &mockTagReader{}

			tt.input.AccountID = accountID.String()
			tt.input.RequestingUserID = ownerID.String()

			accRepo.On("FindByID", mock.Anything, accountID).
				Return(newActiveAccount(t, ownerID, accountID), nil)

			if !tt.skipFindCategory && tt.input.CategoryID != nil && *tt.input.CategoryID != "" {
				catID, parseErr := vo.ParseID(*tt.input.CategoryID)
				if parseErr == nil {
					if tt.findCategoryErr != nil {
						catReader.On("FindVisible", mock.Anything, catID, ownerID).
							Return(nil, tt.findCategoryErr)
					} else {
						catReader.On("FindVisible", mock.Anything, catID, ownerID).
							Return(tt.categoryFix, nil)
					}
				}
			}
			if !tt.skipFindManyVis && len(tt.input.TagIDs) > 0 && len(tt.input.TagIDs) <= tagdomain.MaxTagsPerStatement {
				tagReader.On("FindManyVisible", mock.Anything, mock.Anything, ownerID).
					Return(tt.visibleTags, nil)
			}
			if !tt.skipCreate {
				stmtRepo.On("Create", mock.Anything, mock.AnythingOfType("*statement.Statement"), accountID).
					Return(int64(8000), nil)
			}

			uc := NewCreateUseCase(stmtRepo, accRepo).
				WithCategoryRepo(catReader).
				WithTagRepo(tagReader)

			// Act
			out, execErr := uc.Execute(context.Background(), tt.input)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				if tt.skipCreate {
					stmtRepo.AssertNotCalled(t, "Create")
				}
				if tt.skipFindManyVis {
					tagReader.AssertNotCalled(t, "FindManyVisible")
				}
				if tt.skipFindCategory {
					catReader.AssertNotCalled(t, "FindVisible")
				}
				return
			}
			require.NoError(t, execErr)
			require.NotNil(t, out)
			if tt.wantNullCategory {
				assert.Nil(t, out.Category)
			} else if tt.wantSuccess {
				require.NotNil(t, out.Category)
				assert.Equal(t, tt.wantCategoryName, out.Category.Name)
				assert.Equal(t, tt.wantCategoryType, out.Category.Type)
			}
			require.NotNil(t, out.Tags)
			assert.Len(t, out.Tags, tt.wantTagsCount)
		})
	}
}

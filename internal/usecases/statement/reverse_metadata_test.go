package statement

import (
	"context"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestReverseUseCase_AutoAppliesEstornoCategory exercises the auto-categorization
// rule: every reversal receives the seeded "Estorno" category whose type matches
// the reversal direction (which is the OPPOSITE of the original statement).
func TestReverseUseCase_AutoAppliesEstornoCategory(t *testing.T) {
	ownerID := vo.NewID()
	accountID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Type: accountvo.TypeBankAccount,
		Active: true, Balance: 8000, CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name           string
		originalType   stmtvo.StatementType
		wantReverseTyp stmtvo.StatementType
		wantCategoryID vo.ID
	}{
		{
			name:           "GIVEN debit statement WHEN reverse THEN auto-applies credit Estorno category",
			originalType:   stmtvo.TypeDebit,
			wantReverseTyp: stmtvo.TypeCredit,
			wantCategoryID: categorydomain.SystemCategoryEstornoCreditID,
		},
		{
			name:           "GIVEN credit statement WHEN reverse THEN auto-applies debit Estorno category",
			originalType:   stmtvo.TypeCredit,
			wantReverseTyp: stmtvo.TypeDebit,
			wantCategoryID: categorydomain.SystemCategoryEstornoDebitID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			stmtRepo := &mockRepository{}
			accRepo := &mockAccountRepository{}

			originalID := vo.NewID()
			original := &stmtdomain.Statement{
				ID:           originalID,
				AccountID:    accountID,
				Type:         tt.originalType,
				Amount:       stmtvo.ParseAmount(5000),
				Description:  "Original",
				BalanceAfter: 5000,
				PostedAt:     now,
				CreatedAt:    now,
			}

			accRepo.On("FindByID", mock.Anything, accountID).Return(activeAccount, nil)
			stmtRepo.On("FindByID", mock.Anything, originalID).Return(original, nil)
			stmtRepo.On("HasReversal", mock.Anything, originalID).Return(false, nil)
			stmtRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *stmtdomain.Statement) bool {
				return s.Type == tt.wantReverseTyp &&
					s.CategoryID != nil && *s.CategoryID == tt.wantCategoryID &&
					s.Category != nil && s.Category.Name == "Estorno" &&
					s.Category.Type == tt.wantReverseTyp &&
					len(s.Tags) == 0
			}), accountID).Return(int64(3000), nil)

			uc := NewReverseUseCase(stmtRepo, accRepo)

			// Act
			out, execErr := uc.Execute(context.Background(), dto.ReverseInput{
				StatementID:      originalID.String(),
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
			})

			// Assert
			require.NoError(t, execErr)
			require.NotNil(t, out)
			assert.Equal(t, tt.wantReverseTyp.String(), out.Type)
			require.NotNil(t, out.Category, "reverse must always emit category=Estorno")
			assert.Equal(t, tt.wantCategoryID.String(), out.Category.ID)
			assert.Equal(t, "Estorno", out.Category.Name)
			assert.Equal(t, tt.wantReverseTyp.String(), out.Category.Type)
			assert.NotNil(t, out.Tags)
			assert.Empty(t, out.Tags, "reverse must NOT inherit original tags")
		})
	}
}

package statement

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func TestNewStatement(t *testing.T) {
	accountID := pkgvo.NewID()
	amount, _ := vo.NewAmount(5000)

	s := NewStatement(accountID, vo.TypeCredit, amount, "Salary deposit")

	assert.NotEmpty(t, s.ID)
	assert.Equal(t, accountID, s.AccountID)
	assert.Equal(t, vo.TypeCredit, s.Type)
	assert.Equal(t, amount, s.Amount)
	assert.Equal(t, "Salary deposit", s.Description)
	assert.Nil(t, s.ReferenceID)
	assert.Equal(t, int64(0), s.BalanceAfter)
	assert.NotZero(t, s.CreatedAt)
}

func TestNewReversalStatement(t *testing.T) {
	accountID := pkgvo.NewID()
	referenceID := pkgvo.NewID()
	amount, _ := vo.NewAmount(3000)

	s := NewReversalStatement(accountID, vo.TypeDebit, amount, "Reversal of payment", referenceID)

	assert.NotEmpty(t, s.ID)
	assert.Equal(t, accountID, s.AccountID)
	assert.Equal(t, vo.TypeDebit, s.Type)
	assert.Equal(t, amount, s.Amount)
	assert.Equal(t, "Reversal of payment", s.Description)
	assert.NotNil(t, s.ReferenceID)
	assert.Equal(t, referenceID, *s.ReferenceID)
	assert.Equal(t, int64(0), s.BalanceAfter)
	assert.NotZero(t, s.CreatedAt)
}

func TestStatement_SetBalanceAfter(t *testing.T) {
	accountID := pkgvo.NewID()
	amount, _ := vo.NewAmount(1000)
	s := NewStatement(accountID, vo.TypeCredit, amount, "Test")

	s.SetBalanceAfter(15000)

	assert.Equal(t, int64(15000), s.BalanceAfter)
}

func TestNewStatement_EmptyDescription(t *testing.T) {
	accountID := pkgvo.NewID()
	amount, _ := vo.NewAmount(100)

	s := NewStatement(accountID, vo.TypeDebit, amount, "")

	assert.Equal(t, "", s.Description)
}

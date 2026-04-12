package statement

import (
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Statement represents an immutable financial event (credit or debit) in the ledger.
type Statement struct {
	ID           pkgvo.ID
	AccountID    pkgvo.ID
	Type         vo.StatementType
	Amount       vo.Amount
	Description  string
	ReferenceID  *pkgvo.ID
	BalanceAfter int64
	CreatedAt    time.Time
}

// NewStatement creates a new Statement with default values.
func NewStatement(accountID pkgvo.ID, stmtType vo.StatementType, amount vo.Amount, description string) *Statement {
	return &Statement{
		ID:          pkgvo.NewID(),
		AccountID:   accountID,
		Type:        stmtType,
		Amount:      amount,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// NewReversalStatement creates a reversal Statement linked to the original via ReferenceID.
func NewReversalStatement(accountID pkgvo.ID, stmtType vo.StatementType, amount vo.Amount, description string, referenceID pkgvo.ID) *Statement {
	return &Statement{
		ID:          pkgvo.NewID(),
		AccountID:   accountID,
		Type:        stmtType,
		Amount:      amount,
		Description: description,
		ReferenceID: &referenceID,
		CreatedAt:   time.Now(),
	}
}

// SetBalanceAfter sets the balance snapshot after this statement was applied.
func (s *Statement) SetBalanceAfter(balance int64) {
	s.BalanceAfter = balance
}

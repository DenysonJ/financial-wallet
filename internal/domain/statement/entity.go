package statement

import (
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// TagRef is a lightweight reference to a Tag, embedded in Statement to avoid
// importing the tag domain. Carries ID and Name so output DTOs can be
// hydrated without an extra query.
type TagRef struct {
	ID   pkgvo.ID
	Name string
}

// CategoryRef is a lightweight reference to a Category, embedded in Statement
// for output hydration. Type echoes the statement type by the REQ-8 invariant.
type CategoryRef struct {
	ID   pkgvo.ID
	Name string
	Type vo.StatementType
}

// Statement represents an immutable financial event (credit or debit) in the ledger.
type Statement struct {
	ID           pkgvo.ID
	AccountID    pkgvo.ID
	Type         vo.StatementType
	Amount       vo.Amount
	Description  string
	ReferenceID  *pkgvo.ID
	ExternalID   *string
	BalanceAfter int64
	PostedAt     time.Time
	CreatedAt    time.Time

	// Category assigned to the statement; nil means unclassified.
	// the category type must match Statement.Type.
	CategoryID *pkgvo.ID
	// Hydrated on reads; nil when CategoryID is nil.
	Category *CategoryRef
	// Tag set via statement_tags; always a slice (possibly empty).
	Tags []TagRef
}

// NewStatement creates a new Statement with default values.
func NewStatement(accountID pkgvo.ID, stmtType vo.StatementType, amount vo.Amount, description string) *Statement {
	now := time.Now()
	return &Statement{
		ID:          pkgvo.NewID(),
		AccountID:   accountID,
		Type:        stmtType,
		Amount:      amount,
		Description: description,
		PostedAt:    now,
		CreatedAt:   now,
	}
}

// NewReversalStatement creates a reversal Statement linked to the original via ReferenceID.
func NewReversalStatement(accountID pkgvo.ID, stmtType vo.StatementType, amount vo.Amount, description string, referenceID pkgvo.ID) *Statement {
	now := time.Now()
	return &Statement{
		ID:          pkgvo.NewID(),
		AccountID:   accountID,
		Type:        stmtType,
		Amount:      amount,
		Description: description,
		ReferenceID: &referenceID,
		PostedAt:    now,
		CreatedAt:   now,
	}
}

// NewImportedStatement creates a Statement from an external source (e.g., OFX import).
// PostedAt preserves the original transaction date; CreatedAt records when it was imported.
func NewImportedStatement(accountID pkgvo.ID, stmtType vo.StatementType, amount vo.Amount, description, externalID string, postedAt time.Time) *Statement {
	return &Statement{
		ID:          pkgvo.NewID(),
		AccountID:   accountID,
		Type:        stmtType,
		Amount:      amount,
		Description: description,
		ExternalID:  &externalID,
		PostedAt:    postedAt,
		CreatedAt:   time.Now(),
	}
}

// SetBalanceAfter sets the balance snapshot after this statement was applied.
func (s *Statement) SetBalanceAfter(balance int64) {
	s.BalanceAfter = balance
}

// WithCategory associates a category with the statement. The caller must
// ensure the category type matches Statement.Type before calling.
func (s *Statement) WithCategory(id pkgvo.ID) *Statement {
	s.CategoryID = &id
	return s
}

// WithTags replaces the tag set on the statement.
func (s *Statement) WithTags(refs []TagRef) *Statement {
	s.Tags = refs
	return s
}

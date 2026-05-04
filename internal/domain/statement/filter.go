package statement

import (
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// ListFilter contains filtering and pagination parameters for listing statements.
type ListFilter struct {
	AccountID pkgvo.ID
	Type      *vo.StatementType
	DateFrom  *time.Time
	DateTo    *time.Time
	// CategoryID filters statements with the given category (REQ-13).
	CategoryID *pkgvo.ID
	// TagIDs filters statements that have ANY of the given tag IDs assigned (REQ-13).
	// Empty slice = no tag filter applied.
	TagIDs []pkgvo.ID
	// Page-based pagination (fallback when no cursor)
	Page  int
	Limit int
	// Cursor-based pagination (keyset on posted_at DESC, id DESC)
	CursorPostedAt *time.Time
	CursorID       *pkgvo.ID
}

// UseCursor returns true if cursor-based pagination should be used.
func (f *ListFilter) UseCursor() bool {
	return f.CursorPostedAt != nil && f.CursorID != nil
}

// Normalize applies default values to pagination parameters.
func (f *ListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}

// Offset calculates the SQL offset from page and limit.
func (f *ListFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// ListResult contains the paginated result of statements.
type ListResult struct {
	Statements []*Statement
	Total      int
	Page       int
	Limit      int
	NextCursor string
}

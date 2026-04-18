package account

import (
	"time"

	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// ListFilter contém os parâmetros de filtragem e paginação para listagem de accounts.
type ListFilter struct {
	Page       int
	Limit      int
	UserID     vo.ID
	Name       string
	Type       string
	ActiveOnly bool

	// Cursor-based pagination (keyset on created_at DESC, id DESC). Populated
	// when the client passes an opaque cursor; overrides Page.
	CursorCreatedAt *time.Time
	CursorID        *vo.ID
}

// UseCursor returns true if cursor-based pagination should be used.
func (f *ListFilter) UseCursor() bool {
	return f.CursorCreatedAt != nil && f.CursorID != nil
}

// Normalize aplica valores padrão aos parâmetros de paginação.
func (f *ListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}

// Offset calcula o offset para a query SQL.
func (f *ListFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// ListResult contém o resultado paginado de accounts.
type ListResult struct {
	Accounts   []*Account
	Total      int
	Page       int
	Limit      int
	NextCursor string
}

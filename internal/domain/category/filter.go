package category

import (
	"github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Scope filters categories by origin (system, user-owned, or both).
type Scope string

const (
	// ScopeAll — defaults + user's own (default when filter is omitted).
	ScopeAll Scope = ""
	// ScopeSystem — system defaults only (user_id IS NULL).
	ScopeSystem Scope = "system"
	// ScopeUser — user's own only (defaults excluded).
	ScopeUser Scope = "user"
)

// ListFilter holds parameters for listing visible categories.
type ListFilter struct {
	// UserID identifies the requester.
	UserID pkgvo.ID
	// Type filters by credit/debit; nil means no filter.
	Type *vo.CategoryType
	// Scope filters by origin (system / user / all).
	Scope Scope
}

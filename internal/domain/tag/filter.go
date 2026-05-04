package tag

import pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"

// Scope filters tags by origin (system, user-owned, or both).
type Scope string

const (
	// ScopeAll — defaults + user's own (default when filter is omitted).
	ScopeAll Scope = ""
	// ScopeSystem — system defaults only (user_id IS NULL).
	ScopeSystem Scope = "system"
	// ScopeUser — user's own only (defaults excluded).
	ScopeUser Scope = "user"
)

// ListFilter holds parameters for listing visible tags.
type ListFilter struct {
	// UserID identifies the requester; defaults are always visible when Scope is All or System.
	UserID pkgvo.ID
	// Scope filters by origin (system / user / all).
	Scope Scope
}

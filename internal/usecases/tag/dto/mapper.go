package dto

import (
	"time"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
)

// FromDomain converts a domain Tag to the output DTO.
// Scope is derived from IsSystem() — it is not a DB column.
func FromDomain(t *tagdomain.Tag) TagOutput {
	scope := "user"
	if t.IsSystem() {
		scope = "system"
	}
	return TagOutput{
		ID:        t.ID.String(),
		Name:      t.Name,
		Scope:     scope,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

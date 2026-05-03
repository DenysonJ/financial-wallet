package dto

import (
	"time"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
)

// FromDomain converts a domain Category to the output DTO.
// Scope is derived from IsSystem() — it is not a DB column.
func FromDomain(c *categorydomain.Category) CategoryOutput {
	scope := "user"
	if c.IsSystem() {
		scope = "system"
	}
	return CategoryOutput{
		ID:        c.ID.String(),
		Name:      c.Name,
		Type:      c.Type.String(),
		Scope:     scope,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

package interfaces

import (
	"context"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Repository defines the persistence contract for Category.
//
// Declared in the use-case layer and implemented in infrastructure
// (Dependency Inversion Principle).
type Repository interface {
	// Create persists a new category. Returns ErrCategoryDuplicate when the
	// (user_id, lower(name), type) constraint is violated.
	Create(ctx context.Context, c *categorydomain.Category) error

	// FindByID looks up a category by ID without visibility filtering.
	// Returns ErrCategoryNotFound if missing.
	FindByID(ctx context.Context, id vo.ID) (*categorydomain.Category, error)

	// FindVisible looks up a category by ID applying the visibility filter
	// (`user_id = $userID OR user_id IS NULL`). Returns ErrCategoryNotVisible
	// if not owned by the user and not a default.
	FindVisible(ctx context.Context, id, userID vo.ID) (*categorydomain.Category, error)

	// List returns categories visible to the user (defaults + own), filtered
	// by type/scope per ListFilter.
	List(ctx context.Context, filter categorydomain.ListFilter) ([]*categorydomain.Category, error)

	// Update mutates only Name and updated_at. Type is immutable.
	// Returns ErrCategoryNotFound if the ID does not exist.
	Update(ctx context.Context, c *categorydomain.Category) error

	// Delete removes a category. Returns ErrCategoryNotFound if missing or
	// ErrCategoryInUse on FK violation (still referenced by statements).
	Delete(ctx context.Context, id vo.ID) error

	// CountStatementsUsing returns how many statements reference the category.
	// Used by the Delete use case to surface ErrCategoryInUse before the DELETE.
	CountStatementsUsing(ctx context.Context, id vo.ID) (int, error)
}

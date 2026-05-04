package interfaces

import (
	"context"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Repository defines the persistence contract for Tag.
type Repository interface {
	// Create persists a new tag. Returns ErrTagDuplicate when the
	// (user_id, lower(name)) constraint is violated.
	Create(ctx context.Context, t *tagdomain.Tag) error

	// FindByID looks up a tag by ID without visibility filtering.
	// Returns ErrTagNotFound if missing.
	FindByID(ctx context.Context, id vo.ID) (*tagdomain.Tag, error)

	// FindVisible looks up a tag by ID applying the visibility filter
	// (`user_id = $userID OR user_id IS NULL`). Returns ErrTagNotVisible if
	// not owned by the user and not a default.
	FindVisible(ctx context.Context, id, userID vo.ID) (*tagdomain.Tag, error)

	// FindManyVisible returns only the tags visible to the user (own + defaults).
	// Invalid/missing/not-visible IDs are silently dropped — the caller compares
	// len(in) vs len(out) to detect them.
	FindManyVisible(ctx context.Context, ids []vo.ID, userID vo.ID) ([]*tagdomain.Tag, error)

	// List returns tags visible to the user, filtered by scope.
	List(ctx context.Context, filter tagdomain.ListFilter) ([]*tagdomain.Tag, error)

	// Update mutates only Name and updated_at.
	// Returns ErrTagNotFound if the ID does not exist.
	Update(ctx context.Context, t *tagdomain.Tag) error

	// Delete removes a tag. CASCADE on statement_tags drops the associations —
	// unlike category, a tag in use can still be deleted.
	// Returns ErrTagNotFound if missing.
	Delete(ctx context.Context, id vo.ID) error
}

package interfaces

import (
	"context"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// CategoryReader is the minimal port the statement use case needs from the
// category domain — visibility-scoped reads only. The canonical implementation
// is `repository.CategoryRepository`, which satisfies this interface naturally.
type CategoryReader interface {
	// FindVisible returns the category if visible to the user (own or default);
	// otherwise ErrCategoryNotVisible.
	FindVisible(ctx context.Context, id, userID vo.ID) (*categorydomain.Category, error)
}

// TagReader is the minimal port the statement use case needs from the tag
// domain. It batch-resolves `tag_ids` when creating/updating statements,
// returning only the visible tags (own + defaults) — the caller compares
// lengths to detect invalid/cross-user IDs.
type TagReader interface {
	// FindManyVisible returns only the user's tags or system defaults; tags
	// outside the visibility scope are silently omitted from the response.
	FindManyVisible(ctx context.Context, ids []vo.ID, userID vo.ID) ([]*tagdomain.Tag, error)
}

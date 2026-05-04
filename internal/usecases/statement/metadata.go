package statement

import (
	"context"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/interfaces"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// dedupTagIDs returns a slice of unique IDs preserving first-seen order.
func dedupTagIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// resolveCategory validates ownership/visibility and type-match. Returns the
// domain Category (with type echoed) when categoryIDPtr is non-nil; nil when
// the input did not provide a category.
//
// Errors:
//   - vo.ErrInvalidID — categoryIDPtr is not a valid UUID
//   - categorydomain.ErrCategoryNotVisible — category not owned/visible
//   - categorydomain.ErrCategoryTypeMismatch — category.type ≠ stmtType
func resolveCategory(
	ctx context.Context,
	categoryRepo interfaces.CategoryReader,
	categoryIDPtr *string,
	userID pkgvo.ID,
	stmtType stmtvo.StatementType,
) (*categorydomain.Category, error) {
	if categoryIDPtr == nil || *categoryIDPtr == "" {
		return nil, nil //nolint:nilnil // explicit "no category provided"
	}

	id, parseErr := pkgvo.ParseID(*categoryIDPtr)
	if parseErr != nil {
		return nil, parseErr
	}

	cat, findErr := categoryRepo.FindVisible(ctx, id, userID)
	if findErr != nil {
		return nil, findErr
	}

	if cat.Type.String() != stmtType.String() {
		return nil, categorydomain.ErrCategoryTypeMismatch
	}
	return cat, nil
}

// resolveTags deduplicates the input, enforces MaxTagsPerStatement, and
// validates that every requested tag is visible to the user. Returns the
// resolved domain Tags (so the caller can hydrate TagRef on the entity).
//
// Errors:
//   - tagdomain.ErrTagLimitExceeded — len(unique) > MaxTagsPerStatement
//   - vo.ErrInvalidID — any tag_id is not a valid UUID
//   - tagdomain.ErrTagNotVisible — len(found) < len(unique)
func resolveTags(
	ctx context.Context,
	tagRepo interfaces.TagReader,
	rawIDs []string,
	userID pkgvo.ID,
) ([]*tagdomain.Tag, error) {
	unique := dedupTagIDs(rawIDs)
	if len(unique) == 0 {
		return nil, nil
	}
	if len(unique) > tagdomain.MaxTagsPerStatement {
		return nil, tagdomain.ErrTagLimitExceeded
	}

	parsed := make([]pkgvo.ID, 0, len(unique))
	for _, raw := range unique {
		id, parseErr := pkgvo.ParseID(raw)
		if parseErr != nil {
			return nil, parseErr
		}
		parsed = append(parsed, id)
	}

	found, findErr := tagRepo.FindManyVisible(ctx, parsed, userID)
	if findErr != nil {
		return nil, findErr
	}
	if len(found) != len(unique) {
		return nil, tagdomain.ErrTagNotVisible
	}
	return found, nil
}

// verifyStatementOwnership loads the parent account and verifies the requester
// is the owner. Returns the owning user ID; cross-user attempts return
// stmtdomain.ErrStatementNotFound (no existence oracle).
//
// `requestingUserID` empty string is treated as "service-key/admin context"
// and skips the ownership check (consistent with how other statement use cases
// handle service-to-service traffic).
func verifyStatementOwnership(
	ctx context.Context,
	span trace.Span,
	accountRepo interfaces.AccountRepository,
	stmt *stmtdomain.Statement,
	requestingUserID string,
) (pkgvo.ID, error) {
	account, accErr := accountRepo.FindByID(ctx, stmt.AccountID)
	if accErr != nil {
		span.SetAttributes(attribute.String("app.result", "account_lookup_failed"))
		return "", accErr
	}
	if requestingUserID != "" && account.UserID.String() != requestingUserID {
		span.SetAttributes(attribute.String("app.result", "not_found"))
		return "", stmtdomain.ErrStatementNotFound
	}
	return account.UserID, nil
}

// applyMetadataToStatement attaches resolved category + tags to the entity
// before persistence. Mutates stmt in place.
func applyMetadataToStatement(stmt *stmtdomain.Statement, cat *categorydomain.Category, tags []*tagdomain.Tag) {
	if cat != nil {
		stmt.WithCategory(cat.ID)
		stmt.Category = &stmtdomain.CategoryRef{
			ID:   cat.ID,
			Name: cat.Name,
			Type: stmtvo.ParseStatementType(cat.Type.String()),
		}
	}
	if len(tags) > 0 {
		refs := make([]stmtdomain.TagRef, 0, len(tags))
		for _, tag := range tags {
			refs = append(refs, stmtdomain.TagRef{ID: tag.ID, Name: tag.Name})
		}
		stmt.WithTags(refs)
	}
}

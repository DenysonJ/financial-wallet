package statement

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// UpdateCategoryUseCase implements REQ-11 — set, swap, or clear the category
// of an existing statement.
//
// The category is a metadata classifier and never participates in balance
// computation. The repository's `UpdateCategory` SET clause is restricted to
// `category_id` (+ updated_at), guaranteeing that `amount`, `type`,
// `balance_after` and the append-only chain remain untouched (REQ-11 invariant).
type UpdateCategoryUseCase struct {
	repo         interfaces.Repository
	accountRepo  interfaces.AccountRepository
	categoryRepo interfaces.CategoryReader
}

// NewUpdateCategoryUseCase builds the use case.
func NewUpdateCategoryUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository, categoryRepo interfaces.CategoryReader) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{repo: repo, accountRepo: accountRepo, categoryRepo: categoryRepo}
}

// Execute validates ownership/visibility/type-match and persists the new category.
//
// Paths:
//   - input.CategoryID == nil OR *input.CategoryID == ""  → clear
//   - input.CategoryID != nil                              → set/swap, with validation
//
// Errors (mapped to HTTP in error.go):
//   - ErrStatementNotFound — statement missing OR owned by another user (no oracle)
//   - ErrCategoryNotVisible — category not owned by the user and not a default
//   - ErrCategoryTypeMismatch — category type differs from statement type
func (uc *UpdateCategoryUseCase) Execute(ctx context.Context, input dto.UpdateCategoryInput) (*dto.StatementOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.UpdateCategory")
	defer span.End()

	ctx = injectLogContext(ctx, ActionUpdateCategory)

	stmtID, parseErr := pkgvo.ParseID(input.StatementID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_statement_id"))
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("statement.id", input.StatementID))

	stmt, findErr := uc.repo.FindByID(ctx, stmtID)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "domain_error", "statement update_category failed")
		return nil, findErr
	}

	// Verify ownership through the account.
	requesterID, ownerErr := verifyStatementOwnership(ctx, span, uc.accountRepo, stmt, input.RequestingUserID)
	if ownerErr != nil {
		telemetry.ClassifyError(ctx, span, ownerErr, "domain_error", "statement update_category failed: ownership")
		return nil, ownerErr
	}

	// Resolve target category: empty/nil → clear; otherwise validate visibility + type-match.
	var targetCategoryID *pkgvo.ID
	if input.CategoryID != nil && *input.CategoryID != "" {
		cat, catErr := resolveCategory(ctx, uc.categoryRepo, input.CategoryID, requesterID, stmt.Type)
		if catErr != nil {
			telemetry.ClassifyError(ctx, span, catErr, "domain_error", "statement update_category failed: category")
			return nil, catErr
		}
		targetCategoryID = &cat.ID
		// Hydrate output category ref directly from the resolved category.
		stmt.CategoryID = targetCategoryID
		stmt.Category = &stmtdomain.CategoryRef{
			ID:   cat.ID,
			Name: cat.Name,
			Type: stmt.Type, // by REQ-11 invariant — type-match validated above
		}
	} else {
		stmt.CategoryID = nil
		stmt.Category = nil
	}

	// Persist via the dedicated UpdateCategory query (touches ONLY category_id).
	if updateErr := uc.repo.UpdateCategory(ctx, stmtID, targetCategoryID); updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "statement update_category failed: persist")
		return nil, updateErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "statement category updated", "statement.id", input.StatementID, "category.cleared", targetCategoryID == nil)

	return toOutput(stmt), nil
}

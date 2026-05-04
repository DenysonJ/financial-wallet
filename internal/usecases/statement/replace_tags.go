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

// ReplaceTagsUseCase implements REQ-10 — replace the entire tag set of an
// existing statement (PUT semantics).
//
// Empty list is valid (clears all tags). Visibility/dedup/limit checks reuse
// the shared `resolveTags` helper. The repository's `ReplaceTags` runs the
// DELETE + INSERT atomically and never touches the parent statement row,
// preserving accounting invariants.
type ReplaceTagsUseCase struct {
	repo        interfaces.Repository
	accountRepo interfaces.AccountRepository
	tagRepo     interfaces.TagReader
}

// NewReplaceTagsUseCase builds the use case.
func NewReplaceTagsUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository, tagRepo interfaces.TagReader) *ReplaceTagsUseCase {
	return &ReplaceTagsUseCase{repo: repo, accountRepo: accountRepo, tagRepo: tagRepo}
}

// Execute validates ownership/visibility and replaces the tag set.
//
// Errors:
//   - ErrStatementNotFound — statement missing OR owned by another user
//   - ErrTagNotVisible — one or more tags outside the user's visibility scope
//   - ErrTagLimitExceeded — len(unique(input.TagIDs)) > MaxTagsPerStatement
func (uc *ReplaceTagsUseCase) Execute(ctx context.Context, input dto.ReplaceTagsInput) (*dto.StatementOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.ReplaceTags")
	defer span.End()

	ctx = injectLogContext(ctx, ActionReplaceTags)

	stmtID, parseErr := pkgvo.ParseID(input.StatementID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_statement_id"))
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("statement.id", input.StatementID))

	stmt, findErr := uc.repo.FindByID(ctx, stmtID)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "domain_error", "statement replace_tags failed")
		return nil, findErr
	}

	requesterID, ownerErr := verifyStatementOwnership(ctx, span, uc.accountRepo, stmt, input.RequestingUserID)
	if ownerErr != nil {
		telemetry.ClassifyError(ctx, span, ownerErr, "domain_error", "statement replace_tags failed: ownership")
		return nil, ownerErr
	}

	// Resolve tags (dedup → 10-limit → visibility batch). Empty input is valid (clear).
	tags, tagErr := resolveTags(ctx, uc.tagRepo, input.TagIDs, requesterID)
	if tagErr != nil {
		telemetry.ClassifyError(ctx, span, tagErr, "domain_error", "statement replace_tags failed: tags")
		return nil, tagErr
	}

	// Project resolved tags into the entity for output hydration AND extract IDs for the repo.
	tagIDs := make([]pkgvo.ID, 0, len(tags))
	refs := make([]stmtdomain.TagRef, 0, len(tags))
	for _, t := range tags {
		tagIDs = append(tagIDs, t.ID)
		refs = append(refs, stmtdomain.TagRef{ID: t.ID, Name: t.Name})
	}

	if replaceErr := uc.repo.ReplaceTags(ctx, stmtID, tagIDs); replaceErr != nil {
		telemetry.ClassifyError(ctx, span, replaceErr, "domain_error", "statement replace_tags failed: persist")
		return nil, replaceErr
	}

	// Hydrate output. Note: stmt.Category was loaded by FindByID and is preserved unchanged.
	stmt.Tags = refs

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "statement tags replaced", "statement.id", input.StatementID, "tags.count", len(refs))

	return toOutput(stmt), nil
}

package statement

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// CreateUseCase implements the use case for creating a credit or debit statement.
type CreateUseCase struct {
	repo         interfaces.Repository
	accountRepo  interfaces.AccountRepository
	categoryRepo interfaces.CategoryReader
	tagRepo      interfaces.TagReader
}

// NewCreateUseCase creates a new CreateUseCase instance.
func NewCreateUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository) *CreateUseCase {
	return &CreateUseCase{repo: repo, accountRepo: accountRepo}
}

// WithCategoryRepo attaches the category port. Required when accepting CategoryID
// in CreateInput; if not wired and input has CategoryID, Execute panics with
// a clear error (cf. tests).
func (uc *CreateUseCase) WithCategoryRepo(r interfaces.CategoryReader) *CreateUseCase {
	uc.categoryRepo = r
	return uc
}

// WithTagRepo attaches the tag port. Required when accepting TagIDs in CreateInput.
func (uc *CreateUseCase) WithTagRepo(r interfaces.TagReader) *CreateUseCase {
	uc.tagRepo = r
	return uc
}

// Execute creates a new statement and atomically updates the account balance.
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.StatementOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.Create")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionCreate)

	accountID, parseErr := pkgvo.ParseID(input.AccountID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_account_id"))
		logutil.LogWarn(ctx, "statement creation failed: invalid account ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.AccountID))

	// Find account and verify ownership
	account, findErr := uc.accountRepo.FindByID(ctx, accountID)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "statement creation failed")
		return nil, findErr
	}

	// Ownership denial intentionally surfaces as 404 (no existence oracle);
	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		telemetry.WarnSpan(span, attribute.String("app.result", "not_found"))
		logutil.LogWarn(ctx, "statement creation: access denied", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	if !account.Active {
		telemetry.WarnSpan(span, attribute.String("app.result", "account_not_active"))
		logutil.LogWarn(ctx, "statement creation failed: account not active", "account.id", accountID.String())
		return nil, stmtdomain.ErrAccountNotActive
	}

	// Validate statement type
	stmtType, typeErr := stmtvo.NewStatementType(input.Type)
	if typeErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_type"))
		logutil.LogWarn(ctx, "statement creation failed: invalid type", "error", typeErr.Error())
		return nil, typeErr
	}

	// Validate amount
	amount, amountErr := stmtvo.NewAmount(input.Amount)
	if amountErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_amount"))
		logutil.LogWarn(ctx, "statement creation failed: invalid amount", "error", amountErr.Error())
		return nil, amountErr
	}

	// Parse optional PostedAt (defaults to now inside NewStatement)
	var postedAt time.Time
	if input.PostedAt != "" {
		parsedPostedAt, postedAtErr := time.Parse(time.RFC3339, input.PostedAt)
		if postedAtErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_posted_at"))
			logutil.LogWarn(ctx, "statement creation failed: invalid posted_at", "error", postedAtErr.Error())
			return nil, postedAtErr
		}
		postedAt = parsedPostedAt
	}

	// Resolve metadata (category + tags). UserID for visibility comes from the
	// authenticated requester — categories/tags are always scoped to the owner
	// of the statement (which is the account owner, validated above).
	requesterID := account.UserID
	cat, catErr := resolveCategory(ctx, uc.categoryRepo, input.CategoryID, requesterID, stmtType)
	if catErr != nil {
		telemetry.ClassifyError(ctx, span, catErr, "domain_error", "statement creation failed: category")
		return nil, catErr
	}
	tags, tagErr := resolveTags(ctx, uc.tagRepo, input.TagIDs, requesterID)
	if tagErr != nil {
		telemetry.ClassifyError(ctx, span, tagErr, "domain_error", "statement creation failed: tags")
		return nil, tagErr
	}

	// Create domain entity + attach metadata
	stmt := stmtdomain.NewStatement(accountID, stmtType, amount, input.Description)
	if !postedAt.IsZero() {
		stmt.PostedAt = postedAt
	}
	applyMetadataToStatement(stmt, cat, tags)

	// Persist (transactional: INSERT statement + UPDATE account balance)
	balanceAfter, createErr := uc.repo.Create(ctx, stmt, accountID)
	if createErr != nil {
		telemetry.ClassifyError(ctx, span, createErr, "domain_error", "statement creation failed")
		return nil, createErr
	}
	stmt.SetBalanceAfter(balanceAfter)

	span.SetAttributes(attribute.String("statement.id", stmt.ID.String()))
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "statement created", "statement.id", stmt.ID.String(), "type", stmtType.String())

	return toOutput(stmt), nil
}

// toOutput converts a domain Statement to the output DTO.
//
// Always emits `tags: []` (never null) for client stability.
// `category` is nil when CategoryID is nil and non-nil otherwise.
func toOutput(stmt *stmtdomain.Statement) *dto.StatementOutput {
	output := &dto.StatementOutput{
		ID:           stmt.ID.String(),
		AccountID:    stmt.AccountID.String(),
		Type:         stmt.Type.String(),
		Amount:       stmt.Amount.Int64(),
		Description:  stmt.Description,
		ExternalID:   stmt.ExternalID,
		BalanceAfter: stmt.BalanceAfter,
		PostedAt:     stmt.PostedAt.Format(time.RFC3339),
		CreatedAt:    stmt.CreatedAt.Format(time.RFC3339),
		Tags:         []dto.TagRef{},
	}
	if stmt.ReferenceID != nil {
		ref := stmt.ReferenceID.String()
		output.ReferenceID = &ref
	}
	if stmt.Category != nil {
		output.Category = &dto.CategoryRef{
			ID:   stmt.Category.ID.String(),
			Name: stmt.Category.Name,
			Type: stmt.Category.Type.String(),
		}
	}
	if len(stmt.Tags) > 0 {
		refs := make([]dto.TagRef, 0, len(stmt.Tags))
		for _, t := range stmt.Tags {
			refs = append(refs, dto.TagRef{ID: t.ID.String(), Name: t.Name})
		}
		output.Tags = refs
	}
	return output
}

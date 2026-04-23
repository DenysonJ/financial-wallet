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

// GetUseCase implements the use case for fetching a single statement by ID.
type GetUseCase struct {
	repo        interfaces.Repository
	accountRepo interfaces.AccountRepository
}

// NewGetUseCase creates a new GetUseCase instance.
func NewGetUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository) *GetUseCase {
	return &GetUseCase{repo: repo, accountRepo: accountRepo}
}

// Execute fetches a statement by ID with ownership verification.
func (uc *GetUseCase) Execute(ctx context.Context, input dto.GetInput) (*dto.StatementOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.Get")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionGet)

	statementID, parseErr := pkgvo.ParseID(input.ID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_statement_id"))
		logutil.LogWarn(ctx, "statement get failed: invalid statement ID", "error", parseErr.Error())
		return nil, parseErr
	}

	accountID, accountParseErr := pkgvo.ParseID(input.AccountID)
	if accountParseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_account_id"))
		logutil.LogWarn(ctx, "statement get failed: invalid account ID", "error", accountParseErr.Error())
		return nil, accountParseErr
	}

	span.SetAttributes(
		attribute.String("statement.id", input.ID),
		attribute.String("account.id", input.AccountID),
	)

	// Find account and verify ownership
	account, findAccountErr := uc.accountRepo.FindByID(ctx, accountID)
	if findAccountErr != nil {
		telemetry.ClassifyError(ctx, span, findAccountErr, "not_found", "statement get failed")
		return nil, findAccountErr
	}

	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		telemetry.WarnSpan(span, attribute.String("app.result", "forbidden"))
		logutil.LogWarn(ctx, "statement get forbidden: not owner", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	// Find statement
	stmt, findErr := uc.repo.FindByID(ctx, statementID)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "statement get failed")
		return nil, findErr
	}

	// Verify statement belongs to the account
	if stmt.AccountID != accountID {
		telemetry.WarnSpan(span, attribute.String("app.result", "statement_not_in_account"))
		logutil.LogWarn(ctx, "statement get failed: statement not in account")
		return nil, stmtdomain.ErrStatementNotFound
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "statement retrieved", "statement.id", statementID.String())

	return toOutput(stmt), nil
}

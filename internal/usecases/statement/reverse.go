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

// ReverseUseCase implements the use case for reversing an existing statement.
type ReverseUseCase struct {
	repo        interfaces.Repository
	accountRepo interfaces.AccountRepository
}

// NewReverseUseCase creates a new ReverseUseCase instance.
func NewReverseUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository) *ReverseUseCase {
	return &ReverseUseCase{repo: repo, accountRepo: accountRepo}
}

// Execute reverses an existing statement by creating an opposite-type statement.
func (uc *ReverseUseCase) Execute(ctx context.Context, input dto.ReverseInput) (*dto.StatementOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.Reverse")
	defer span.End()

	ctx = injectLogContext(ctx, ActionReverse)

	statementID, parseErr := pkgvo.ParseID(input.StatementID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_statement_id"))
		logutil.LogWarn(ctx, "statement reversal failed: invalid statement ID", "error", parseErr.Error())
		return nil, parseErr
	}

	accountID, accountParseErr := pkgvo.ParseID(input.AccountID)
	if accountParseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_account_id"))
		logutil.LogWarn(ctx, "statement reversal failed: invalid account ID", "error", accountParseErr.Error())
		return nil, accountParseErr
	}

	span.SetAttributes(
		attribute.String("statement.id", input.StatementID),
		attribute.String("account.id", input.AccountID),
	)

	// Find account and verify ownership
	account, findAccountErr := uc.accountRepo.FindByID(ctx, accountID)
	if findAccountErr != nil {
		telemetry.ClassifyError(ctx, span, findAccountErr, "not_found", "statement reversal failed")
		return nil, findAccountErr
	}

	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		telemetry.WarnSpan(span, attribute.String("app.result", "forbidden"))
		logutil.LogWarn(ctx, "statement reversal forbidden: not owner", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	if !account.Active {
		telemetry.WarnSpan(span, attribute.String("app.result", "account_not_active"))
		logutil.LogWarn(ctx, "statement reversal failed: account not active", "account.id", accountID.String())
		return nil, stmtdomain.ErrAccountNotActive
	}

	// Find original statement
	original, findErr := uc.repo.FindByID(ctx, statementID)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "statement reversal failed")
		return nil, findErr
	}

	// Verify statement belongs to this account
	if original.AccountID != accountID {
		telemetry.WarnSpan(span, attribute.String("app.result", "statement_not_in_account"))
		logutil.LogWarn(ctx, "statement reversal failed: statement not in account")
		return nil, stmtdomain.ErrStatementNotFound
	}

	// Check if already reversed
	hasReversal, reversalErr := uc.repo.HasReversal(ctx, statementID)
	if reversalErr != nil {
		telemetry.FailSpan(span, reversalErr, "statement reversal failed")
		logutil.LogError(ctx, "statement reversal failed: check reversal error", "error", reversalErr.Error())
		return nil, reversalErr
	}
	if hasReversal {
		telemetry.WarnSpan(span, attribute.String("app.result", "already_reversed"))
		logutil.LogWarn(ctx, "statement reversal failed: already reversed", "statement.id", statementID.String())
		return nil, stmtdomain.ErrAlreadyReversed
	}

	// Create reversal statement with opposite type
	description := input.Description
	if description == "" {
		description = "Reversal of statement " + statementID.String()
	}

	reversal := stmtdomain.NewReversalStatement(
		accountID,
		original.Type.Opposite(),
		original.Amount,
		description,
		statementID,
	)

	// Persist (transactional)
	balanceAfter, createErr := uc.repo.Create(ctx, reversal, accountID)
	if createErr != nil {
		telemetry.ClassifyError(ctx, span, createErr, "domain_error", "statement reversal failed")
		return nil, createErr
	}
	reversal.SetBalanceAfter(balanceAfter)

	span.SetAttributes(attribute.String("reversal.id", reversal.ID.String()))
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "statement reversed", "reversal.id", reversal.ID.String(), "original.id", statementID.String())

	return toOutput(reversal), nil
}

package statement

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
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

	// Validate IDs
	statementID, parseErr := pkgvo.ParseID(input.StatementID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "statement reversal failed: invalid statement ID", "error", parseErr.Error())
		return nil, parseErr
	}

	accountID, accountParseErr := pkgvo.ParseID(input.AccountID)
	if accountParseErr != nil {
		span.SetStatus(otelcodes.Error, accountParseErr.Error())
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
		span.SetStatus(otelcodes.Error, findAccountErr.Error())
		logutil.LogWarn(ctx, "statement reversal failed: account not found", "error", findAccountErr.Error())
		return nil, findAccountErr
	}

	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		span.SetStatus(otelcodes.Error, "forbidden")
		logutil.LogWarn(ctx, "statement reversal forbidden: not owner", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	if !account.Active {
		span.SetStatus(otelcodes.Error, "account not active")
		logutil.LogWarn(ctx, "statement reversal failed: account not active", "account.id", accountID.String())
		return nil, stmtdomain.ErrAccountNotActive
	}

	// Find original statement
	original, findErr := uc.repo.FindByID(ctx, statementID)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "statement reversal failed: statement not found", "error", findErr.Error())
		return nil, findErr
	}

	// Verify statement belongs to this account
	if original.AccountID != accountID {
		span.SetStatus(otelcodes.Error, "statement not in account")
		logutil.LogWarn(ctx, "statement reversal failed: statement not in account")
		return nil, stmtdomain.ErrStatementNotFound
	}

	// Check if already reversed
	hasReversal, reversalErr := uc.repo.HasReversal(ctx, statementID)
	if reversalErr != nil {
		span.SetStatus(otelcodes.Error, reversalErr.Error())
		logutil.LogError(ctx, "statement reversal failed: check reversal error", "error", reversalErr.Error())
		return nil, reversalErr
	}
	if hasReversal {
		span.SetStatus(otelcodes.Error, "already reversed")
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
		span.SetStatus(otelcodes.Error, createErr.Error())
		logutil.LogError(ctx, "statement reversal failed: repository error", "error", createErr.Error())
		return nil, createErr
	}
	reversal.SetBalanceAfter(balanceAfter)

	span.SetAttributes(attribute.String("reversal.id", reversal.ID.String()))
	logutil.LogInfo(ctx, "statement reversed", "reversal.id", reversal.ID.String(), "original.id", statementID.String())

	return toOutput(reversal), nil
}

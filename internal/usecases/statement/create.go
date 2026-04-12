package statement

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// CreateUseCase implements the use case for creating a credit or debit statement.
type CreateUseCase struct {
	repo        interfaces.Repository
	accountRepo interfaces.AccountRepository
}

// NewCreateUseCase creates a new CreateUseCase instance.
func NewCreateUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository) *CreateUseCase {
	return &CreateUseCase{repo: repo, accountRepo: accountRepo}
}

// Execute creates a new statement and atomically updates the account balance.
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.StatementOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.Create")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionCreate)

	// Validate AccountID
	accountID, parseErr := pkgvo.ParseID(input.AccountID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "statement creation failed: invalid account ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.AccountID))

	// Find account and verify ownership
	account, findErr := uc.accountRepo.FindByID(ctx, accountID)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "statement creation failed: account not found", "error", findErr.Error())
		return nil, findErr
	}

	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		span.SetStatus(otelcodes.Error, "forbidden")
		logutil.LogWarn(ctx, "statement creation forbidden: not owner", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	if !account.Active {
		span.SetStatus(otelcodes.Error, "account not active")
		logutil.LogWarn(ctx, "statement creation failed: account not active", "account.id", accountID.String())
		return nil, stmtdomain.ErrAccountNotActive
	}

	// Validate statement type
	stmtType, typeErr := stmtvo.NewStatementType(input.Type)
	if typeErr != nil {
		span.SetStatus(otelcodes.Error, typeErr.Error())
		logutil.LogWarn(ctx, "statement creation failed: invalid type", "error", typeErr.Error())
		return nil, typeErr
	}

	// Validate amount
	amount, amountErr := stmtvo.NewAmount(input.Amount)
	if amountErr != nil {
		span.SetStatus(otelcodes.Error, amountErr.Error())
		logutil.LogWarn(ctx, "statement creation failed: invalid amount", "error", amountErr.Error())
		return nil, amountErr
	}

	// Create domain entity
	stmt := stmtdomain.NewStatement(accountID, stmtType, amount, input.Description)

	// Persist (transactional: INSERT statement + UPDATE account balance)
	balanceAfter, createErr := uc.repo.Create(ctx, stmt, accountID)
	if createErr != nil {
		span.SetStatus(otelcodes.Error, createErr.Error())
		logutil.LogError(ctx, "statement creation failed: repository error", "error", createErr.Error())
		return nil, createErr
	}
	stmt.SetBalanceAfter(balanceAfter)

	span.SetAttributes(attribute.String("statement.id", stmt.ID.String()))
	logutil.LogInfo(ctx, "statement created", "statement.id", stmt.ID.String(), "type", stmtType.String())

	return toOutput(stmt), nil
}

// toOutput converts a domain Statement to the output DTO.
func toOutput(stmt *stmtdomain.Statement) *dto.StatementOutput {
	output := &dto.StatementOutput{
		ID:           stmt.ID.String(),
		AccountID:    stmt.AccountID.String(),
		Type:         stmt.Type.String(),
		Amount:       stmt.Amount.Int64(),
		Description:  stmt.Description,
		BalanceAfter: stmt.BalanceAfter,
		CreatedAt:    stmt.CreatedAt.Format(time.RFC3339),
	}
	if stmt.ReferenceID != nil {
		ref := stmt.ReferenceID.String()
		output.ReferenceID = &ref
	}
	return output
}

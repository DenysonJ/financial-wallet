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

// ListUseCase implements the use case for listing statements by account.
type ListUseCase struct {
	repo        interfaces.Repository
	accountRepo interfaces.AccountRepository
}

// NewListUseCase creates a new ListUseCase instance.
func NewListUseCase(repo interfaces.Repository, accountRepo interfaces.AccountRepository) *ListUseCase {
	return &ListUseCase{repo: repo, accountRepo: accountRepo}
}

// Execute lists statements for an account with optional filters and pagination.
func (uc *ListUseCase) Execute(ctx context.Context, input dto.ListInput) (*dto.ListOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Statement.List")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionList)

	// Validate AccountID
	accountID, parseErr := pkgvo.ParseID(input.AccountID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "statement list failed: invalid account ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.AccountID))

	// Find account and verify ownership
	account, findErr := uc.accountRepo.FindByID(ctx, accountID)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "statement list failed: account not found", "error", findErr.Error())
		return nil, findErr
	}

	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		span.SetStatus(otelcodes.Error, "forbidden")
		logutil.LogWarn(ctx, "statement list forbidden: not owner", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	// Build filter
	filter := stmtdomain.ListFilter{
		AccountID: accountID,
		Page:      input.Page,
		Limit:     input.Limit,
	}

	// Parse optional type filter
	if input.Type != "" {
		stmtType, typeErr := stmtvo.NewStatementType(input.Type)
		if typeErr != nil {
			span.SetStatus(otelcodes.Error, typeErr.Error())
			logutil.LogWarn(ctx, "statement list failed: invalid type filter", "error", typeErr.Error())
			return nil, typeErr
		}
		filter.Type = &stmtType
	}

	// Parse optional date filters
	if input.DateFrom != "" {
		dateFrom, dateFromErr := time.Parse(time.RFC3339, input.DateFrom)
		if dateFromErr != nil {
			span.SetStatus(otelcodes.Error, dateFromErr.Error())
			logutil.LogWarn(ctx, "statement list failed: invalid date_from", "error", dateFromErr.Error())
			return nil, dateFromErr
		}
		filter.DateFrom = &dateFrom
	}
	if input.DateTo != "" {
		dateTo, dateToErr := time.Parse(time.RFC3339, input.DateTo)
		if dateToErr != nil {
			span.SetStatus(otelcodes.Error, dateToErr.Error())
			logutil.LogWarn(ctx, "statement list failed: invalid date_to", "error", dateToErr.Error())
			return nil, dateToErr
		}
		filter.DateTo = &dateTo
	}

	filter.Normalize()

	// List statements
	result, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		span.SetStatus(otelcodes.Error, listErr.Error())
		logutil.LogError(ctx, "statement list failed: repository error", "error", listErr.Error())
		return nil, listErr
	}

	// Build output
	data := make([]dto.StatementOutput, 0, len(result.Statements))
	for _, stmt := range result.Statements {
		data = append(data, *toOutput(stmt))
	}

	totalPages := 0
	if result.Limit > 0 {
		totalPages = (result.Total + result.Limit - 1) / result.Limit
	}

	logutil.LogInfo(ctx, "statements listed", "account.id", accountID.String(), "count", len(data))

	return &dto.ListOutput{
		Data: data,
		Pagination: dto.PaginationOutput{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

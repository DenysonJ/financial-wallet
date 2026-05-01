package statement

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
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

	accountID, parseErr := pkgvo.ParseID(input.AccountID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_account_id"))
		logutil.LogWarn(ctx, "statement list failed: invalid account ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(attribute.String("account.id", input.AccountID))

	// Find account and verify ownership
	account, findErr := uc.accountRepo.FindByID(ctx, accountID)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "statement list failed")
		return nil, findErr
	}

	// Ownership denial intentionally surfaces as 404 (no existence oracle);
	if input.RequestingUserID != "" && account.UserID.String() != input.RequestingUserID {
		telemetry.WarnSpan(span, attribute.String("app.result", "not_found"))
		logutil.LogWarn(ctx, "statement list: access denied", "account.id", accountID.String())
		return nil, stmtdomain.ErrStatementNotFound
	}

	// Build filter
	filter := stmtdomain.ListFilter{
		AccountID: accountID,
		Page:      input.Page,
		Limit:     input.Limit,
	}

	// Parse optional cursor (overrides page-based pagination)
	if input.Cursor != "" {
		cursorPostedAt, cursorID, cursorErr := decodeCursor(input.Cursor)
		if cursorErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_cursor"))
			logutil.LogWarn(ctx, "statement list failed: invalid cursor", "error", cursorErr.Error())
			return nil, cursorErr
		}
		filter.CursorPostedAt = &cursorPostedAt
		filter.CursorID = &cursorID
	}

	// Parse optional type filter
	if input.Type != "" {
		stmtType, typeErr := stmtvo.NewStatementType(input.Type)
		if typeErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_type"))
			logutil.LogWarn(ctx, "statement list failed: invalid type filter", "error", typeErr.Error())
			return nil, typeErr
		}
		filter.Type = &stmtType
	}

	// Parse optional date filters
	if input.DateFrom != "" {
		dateFrom, dateFromErr := time.Parse(time.RFC3339, input.DateFrom)
		if dateFromErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_date_from"))
			logutil.LogWarn(ctx, "statement list failed: invalid date_from", "error", dateFromErr.Error())
			return nil, dateFromErr
		}
		filter.DateFrom = &dateFrom
	}
	if input.DateTo != "" {
		dateTo, dateToErr := time.Parse(time.RFC3339, input.DateTo)
		if dateToErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_date_to"))
			logutil.LogWarn(ctx, "statement list failed: invalid date_to", "error", dateToErr.Error())
			return nil, dateToErr
		}
		filter.DateTo = &dateTo
	}

	filter.Normalize()

	// List operations return only infrastructure errors — no expected domain
	// sentinels apply, so FailSpan is used directly without IsExpected guard.
	result, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		telemetry.FailSpan(span, listErr, "statement list failed")
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

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "statements listed", "account.id", accountID.String(), "count", len(data))

	pagination := dto.PaginationOutput{
		Page:       result.Page,
		Limit:      result.Limit,
		Total:      result.Total,
		TotalPages: totalPages,
	}
	if result.NextCursor != "" {
		pagination.NextCursor = &result.NextCursor
	}

	return &dto.ListOutput{
		Data:       data,
		Pagination: pagination,
	}, nil
}

// decodeCursor parses an opaque cursor back into posted_at and id.
func decodeCursor(cursor string) (time.Time, pkgvo.ID, error) {
	var zeroID pkgvo.ID

	raw, decodeErr := base64.URLEncoding.DecodeString(cursor)
	if decodeErr != nil {
		return time.Time{}, zeroID, fmt.Errorf("invalid cursor: %w", decodeErr)
	}

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, zeroID, fmt.Errorf("invalid cursor format")
	}

	postedAt, timeErr := time.Parse(time.RFC3339Nano, parts[0])
	if timeErr != nil {
		return time.Time{}, zeroID, fmt.Errorf("invalid cursor timestamp: %w", timeErr)
	}

	id, idErr := pkgvo.ParseID(parts[1])
	if idErr != nil {
		return time.Time{}, zeroID, fmt.Errorf("invalid cursor ID: %w", idErr)
	}

	return postedAt, id, nil
}

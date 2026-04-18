package handler

import (
	"errors"
	"net/http"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/pkg/apperror"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/ofx"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ErrorResponse represents the standard error response for Swagger documentation.
type ErrorResponse struct {
	Errors struct {
		Message string `json:"message" example:"error description"`
	} `json:"errors"`
}

// codeToStatus maps AppError codes to HTTP status codes.
var codeToStatus = map[string]int{
	apperror.CodeInvalidRequest:  http.StatusBadRequest,
	apperror.CodeValidationError: http.StatusBadRequest,
	apperror.CodeNotFound:        http.StatusNotFound,
	apperror.CodeConflict:        http.StatusConflict,
	apperror.CodeUnauthorized:    http.StatusUnauthorized,
	apperror.CodeForbidden:       http.StatusForbidden,
	apperror.CodeInternalError:   http.StatusInternalServerError,
}

// domainErrorMapping defines how pure domain errors translate to HTTP responses.
type domainErrorMapping struct {
	Status  int
	Code    string
	Message string
}

// domainErrors maps domain sentinel errors to their HTTP representation.
// This is the single source of truth for domain error-to-HTTP translation.
var domainErrors = []struct {
	err     error
	mapping domainErrorMapping
}{
	{uservo.ErrInvalidEmail, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid email"}},
	{vo.ErrInvalidID, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid ID"}},
	{uservo.ErrPasswordTooShort, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "password must be at least 8 characters"}},
	{uservo.ErrPasswordNoLetter, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "password must contain at least one letter"}},
	{uservo.ErrPasswordNoNumber, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "password must contain at least one number"}},
	{uservo.ErrPasswordNoSpecial, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "password must contain at least one special character"}},
	{userdomain.ErrPasswordMismatch, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "passwords do not match"}},
	{userdomain.ErrPasswordAlreadySet, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "password already set"}},
	{userdomain.ErrInvalidCredentials, domainErrorMapping{http.StatusUnauthorized, apperror.CodeUnauthorized, "invalid credentials"}},
	{uservo.ErrInvalidPassword, domainErrorMapping{http.StatusUnauthorized, apperror.CodeUnauthorized, "invalid password"}},
	{userdomain.ErrUserInactive, domainErrorMapping{http.StatusUnauthorized, apperror.CodeUnauthorized, "invalid credentials"}},
	{userdomain.ErrUserNotFound, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "user not found"}},
	{roledomain.ErrRoleNotFound, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "role not found"}},
	{roledomain.ErrDuplicateRoleName, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "role name already exists"}},
	{roledomain.ErrRoleAlreadyAssigned, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "role already assigned to user"}},
	{roledomain.ErrRoleNotAssigned, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "role not assigned to user"}},
	{roledomain.ErrForbidden, domainErrorMapping{http.StatusForbidden, apperror.CodeForbidden, "forbidden"}},
	// Account domain errors
	{accountvo.ErrInvalidAccountType, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid account type"}},
	{accountdomain.ErrAccountNotFound, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "account not found"}},
	// Statement domain errors
	{stmtvo.ErrInvalidStatementType, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid statement type"}},
	{stmtvo.ErrInvalidAmount, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "amount must be greater than zero"}},
	{stmtdomain.ErrStatementNotFound, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "statement not found"}},
	{stmtdomain.ErrAlreadyReversed, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "statement already reversed"}},
	{stmtdomain.ErrAccountNotActive, domainErrorMapping{http.StatusUnprocessableEntity, apperror.CodeValidationError, "account is not active"}},
	// OFX parser errors
	{ofx.ErrInvalidFormat, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid OFX file format"}},
	{ofx.ErrNoTransactions, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "no transactions found in OFX file"}},
	{ofx.ErrInvalidAmount, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid amount in OFX file"}},
	{ofx.ErrInvalidDate, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid date in OFX file"}},
}

// HandleError handles errors in a centralized and consistent way.
// It supports AppError (structured) and falls back to domain error translation.
func HandleError(c *gin.Context, span trace.Span, err error) {
	// 1. Try AppError first (structured errors from use cases)
	if appErr, ok2 := errors.AsType[*apperror.AppError](err); ok2 {
		status, ok := codeToStatus[appErr.Code]
		if !ok {
			status = http.StatusInternalServerError
		}
		span.SetStatus(codes.Error, appErr.Code)
		if status >= 500 {
			span.RecordError(err)
		}
		httpgin.SendError(c, status, appErr.Message)
		return
	}

	// 2. Fallback: translate domain errors to HTTP
	status, code, message := translateError(err)

	span.SetStatus(codes.Error, code)
	if status >= 500 {
		span.RecordError(err)
	}

	httpgin.SendError(c, status, message)
}

// translateError maps domain errors to HTTP responses by looking up domainErrors.
func translateError(err error) (status int, code, message string) {
	for _, entry := range domainErrors {
		if errors.Is(err, entry.err) {
			return entry.mapping.Status, entry.mapping.Code, entry.mapping.Message
		}
	}
	return http.StatusInternalServerError, apperror.CodeInternalError, "internal server error"
}

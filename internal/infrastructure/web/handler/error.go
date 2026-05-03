package handler

import (
	"errors"
	"net/http"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/pkg/apperror"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/ofx"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/gin-gonic/gin"
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
	// Category domain errors
	{categoryvo.ErrInvalidCategoryType, domainErrorMapping{http.StatusBadRequest, apperror.CodeInvalidRequest, "invalid category type"}},
	{categorydomain.ErrCategoryInvalidName, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "category name must not be empty"}},
	{categorydomain.ErrCategoryNotFound, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "category not found"}},
	{categorydomain.ErrCategoryDuplicate, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "category already exists"}},
	{categorydomain.ErrCategoryReadOnly, domainErrorMapping{http.StatusForbidden, apperror.CodeForbidden, "system category is read-only"}},
	{categorydomain.ErrCategoryInUse, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "category is in use by one or more statements"}},
	{categorydomain.ErrCategoryTypeMismatch, domainErrorMapping{http.StatusUnprocessableEntity, apperror.CodeValidationError, "category type does not match statement type"}},
	{categorydomain.ErrCategoryNotVisible, domainErrorMapping{http.StatusUnprocessableEntity, apperror.CodeValidationError, "category is not visible to the user"}},
	// Tag domain errors
	{tagdomain.ErrTagInvalidName, domainErrorMapping{http.StatusBadRequest, apperror.CodeValidationError, "tag name must not be empty"}},
	{tagdomain.ErrTagNotFound, domainErrorMapping{http.StatusNotFound, apperror.CodeNotFound, "tag not found"}},
	{tagdomain.ErrTagDuplicate, domainErrorMapping{http.StatusConflict, apperror.CodeConflict, "tag already exists"}},
	{tagdomain.ErrTagReadOnly, domainErrorMapping{http.StatusForbidden, apperror.CodeForbidden, "system tag is read-only"}},
	{tagdomain.ErrTagNotVisible, domainErrorMapping{http.StatusUnprocessableEntity, apperror.CodeValidationError, "tag is not visible to the user"}},
	{tagdomain.ErrTagLimitExceeded, domainErrorMapping{http.StatusUnprocessableEntity, apperror.CodeValidationError, "tag limit per statement exceeded"}},
}

// init publishes every sentinel listed in domainErrors to
// apperror.DomainSentinels so pkg/telemetry.IsExpected can classify span
// errors using the same source of truth as the HTTP translation.
func init() {
	sentinels := make([]error, 0, len(domainErrors))
	for _, entry := range domainErrors {
		sentinels = append(sentinels, entry.err)
	}
	apperror.Register(sentinels...)
}

// HandleError handles errors in a centralized and consistent way.
//
// Precedence is intentional: a domain-sentinel match (via errors.Is, which
// traverses AppError.Unwrap) always wins over AppError.Message. This keeps
// client-facing messages confined to the static strings in domainErrors —
// even if an AppError further up the chain carries verbose text from an
// external parser (e.g. OFX/XML decoder). See security review SEC-H-3.
//
// Span manipulation (RecordError/SetStatus) happens in the use case layer via
// pkg/telemetry.FailSpan / WarnSpan — this function only translates the error
// to an HTTP response.
func HandleError(c *gin.Context, err error) {
	// 1. Domain sentinel match — safe static message wins.
	if status, _, message, ok := translateDomainError(err); ok {
		httpgin.SendError(c, status, message)
		return
	}

	// 2. Structured AppError with no sentinel match.
	if appErr, ok2 := errors.AsType[*apperror.AppError](err); ok2 {
		status, ok := codeToStatus[appErr.Code]
		if !ok {
			status = http.StatusInternalServerError
		}
		httpgin.SendError(c, status, appErr.Message)
		return
	}

	// 3. Unknown error — default to 500 with a generic message.
	httpgin.SendError(c, http.StatusInternalServerError, "internal server error")
}

// translateDomainError looks up err in domainErrors via errors.Is. Returns
// ok=false when no sentinel matches; the caller is responsible for choosing
// the fallback behavior.
func translateDomainError(err error) (status int, code, message string, ok bool) {
	for _, entry := range domainErrors {
		if errors.Is(err, entry.err) {
			return entry.mapping.Status, entry.mapping.Code, entry.mapping.Message, true
		}
	}
	return 0, "", "", false
}

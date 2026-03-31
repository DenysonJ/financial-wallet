package handler

import (
	"net/http"

	useruc "github.com/DenysonJ/financial-wallet/internal/usecases/user"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// PasswordHandler agrupa os handlers de gerenciamento de senha.
type PasswordHandler struct {
	SetPasswordUC    *useruc.SetPasswordUseCase
	ChangePasswordUC *useruc.ChangePasswordUseCase
}

// NewPasswordHandler cria um novo PasswordHandler.
func NewPasswordHandler(
	setPasswordUC *useruc.SetPasswordUseCase,
	changePasswordUC *useruc.ChangePasswordUseCase,
) *PasswordHandler {
	return &PasswordHandler{
		SetPasswordUC:    setPasswordUC,
		ChangePasswordUC: changePasswordUC,
	}
}

// SetPassword godoc
// @Summary      Set user password
// @Description  Set initial password for a user that does not have one yet. Protected by Service Key (not JWT) to avoid deadlock: user needs a password to login, but needs to login to get JWT.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body dto.SetPasswordInput true "User ID, password and confirmation"
// @Success      204
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /users/password [post]
func (h *PasswordHandler) SetPassword(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "PasswordHandler.SetPassword")
	defer span.End()

	var req dto.SetPasswordInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid request body")
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	execErr := h.SetPasswordUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	c.Status(http.StatusNoContent)
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Change password for an authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body dto.ChangePasswordInput true "Current and new password"
// @Success      204
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/password [put]
func (h *PasswordHandler) ChangePassword(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "PasswordHandler.ChangePassword")
	defer span.End()

	var req dto.ChangePasswordInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid request body")
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// User ID from JWT context
	userID, exists := c.Get("user_id")
	if !exists {
		span.SetStatus(codes.Error, "unauthorized")
		httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	req.UserID = userID.(string)

	execErr := h.ChangePasswordUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	c.Status(http.StatusNoContent)
}

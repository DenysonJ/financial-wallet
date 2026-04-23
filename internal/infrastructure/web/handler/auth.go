package handler

import (
	"net/http"

	authuc "github.com/DenysonJ/financial-wallet/internal/usecases/auth"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

// AuthHandler agrupa os handlers de autenticação.
type AuthHandler struct {
	LoginUC   *authuc.LoginUseCase
	RefreshUC *authuc.RefreshUseCase
}

// NewAuthHandler cria um novo AuthHandler.
func NewAuthHandler(loginUC *authuc.LoginUseCase, refreshUC *authuc.RefreshUseCase) *AuthHandler {
	return &AuthHandler{
		LoginUC:   loginUC,
		RefreshUC: refreshUC,
	}
}

// Login godoc
// @Summary      User login
// @Description  Authenticate with email and password, returns JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginInput true "Login credentials"
// @Success      200  {object}  dto.LoginOutput
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "AuthHandler.Login")
	defer span.End()

	var req dto.LoginInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	res, execErr := h.LoginUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Refresh godoc
// @Summary      Refresh tokens
// @Description  Exchange a valid refresh token for a new token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshInput true "Refresh token"
// @Success      200  {object}  dto.RefreshOutput
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "AuthHandler.Refresh")
	defer span.End()

	var req dto.RefreshInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	res, execErr := h.RefreshUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

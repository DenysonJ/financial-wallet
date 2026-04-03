package handler

import (
	"net/http"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	accountuc "github.com/DenysonJ/financial-wallet/internal/usecases/account"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AccountHandler agrupa todos os handlers relacionados a Account.
type AccountHandler struct {
	CreateUC *accountuc.CreateUseCase
	GetUC    *accountuc.GetUseCase
	ListUC   *accountuc.ListUseCase
	UpdateUC *accountuc.UpdateUseCase
	DeleteUC *accountuc.DeleteUseCase
}

// NewAccountHandler cria um novo AccountHandler com todos os use cases.
func NewAccountHandler(
	createUC *accountuc.CreateUseCase,
	getUC *accountuc.GetUseCase,
	listUC *accountuc.ListUseCase,
	updateUC *accountuc.UpdateUseCase,
	deleteUC *accountuc.DeleteUseCase,
) *AccountHandler {
	return &AccountHandler{
		CreateUC: createUC,
		GetUC:    getUC,
		ListUC:   listUC,
		UpdateUC: updateUC,
		DeleteUC: deleteUC,
	}
}

// getJWTUserID extracts the user ID from JWT context.
func getJWTUserID(c *gin.Context) string {
	userID, _ := c.Get(middleware.ContextKeyUserID)
	userIDStr, _ := userID.(string)
	return userIDStr
}

// Create godoc
// @Summary      Create a new account
// @Description  Create a new financial account for the authenticated user
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateInput true "Account info"
// @Success      201  {object}  dto.CreateOutput
// @Failure      400  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts [post]
func (h *AccountHandler) Create(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "AccountHandler.Create")
	defer span.End()

	var req dto.CreateInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid request body")
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Set UserID from JWT context
	req.UserID = getJWTUserID(c)

	span.SetAttributes(attribute.String("account.name", req.Name), attribute.String("account.type", req.Type))

	res, execErr := h.CreateUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	span.SetAttributes(attribute.String("account.id", res.ID))
	httpgin.SendSuccess(c, http.StatusCreated, res)
}

// GetByID godoc
// @Summary      Get an account by ID
// @Description  Get account details by unique ID
// @Tags         accounts
// @Produce      json
// @Param        id   path      string  true  "Account ID"
// @Success      200  {object}  dto.GetOutput
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id} [get]
func (h *AccountHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "AccountHandler.GetByID")
	defer span.End()

	id := c.Param("id")
	span.SetAttributes(attribute.String("account.id", id))

	res, execErr := h.GetUC.Execute(ctx, dto.GetInput{ID: id})
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	// Ownership check: account belongs to user, or admin/service-key
	if !isAdminOrOwner(c, res.UserID) {
		span.SetStatus(codes.Error, "forbidden")
		httpgin.SendError(c, http.StatusForbidden, "forbidden")
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// List godoc
// @Summary      List accounts
// @Description  Get a paginated list of the authenticated user's accounts
// @Tags         accounts
// @Produce      json
// @Param        page        query     int     false  "Page number"
// @Param        limit       query     int     false  "Items per page"
// @Param        name        query     string  false  "Filter by name"
// @Param        type        query     string  false  "Filter by type"
// @Param        active_only query     bool    false  "Filter by active status"
// @Success      200         {object}  dto.ListOutput
// @Failure      429         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts [get]
func (h *AccountHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "AccountHandler.List")
	defer span.End()

	var req dto.ListInput
	if bindErr := c.ShouldBindQuery(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid query parameters")
		httpgin.SendError(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	// Scope to authenticated user's accounts
	req.UserID = getJWTUserID(c)

	span.SetAttributes(
		attribute.Int("filter.page", req.Page),
		attribute.Int("filter.limit", req.Limit),
	)

	res, execErr := h.ListUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	span.SetAttributes(attribute.Int("result.total", res.Pagination.Total))
	httpgin.SendSuccessWithMeta(c, http.StatusOK, res.Data, res.Pagination, nil)
}

// Update godoc
// @Summary      Update an account
// @Description  Update account details by ID
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        id       path      string           true  "Account ID"
// @Param        request  body      dto.UpdateInput  true  "Update info"
// @Success      200      {object}  dto.UpdateOutput
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      429      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id} [put]
func (h *AccountHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "AccountHandler.Update")
	defer span.End()

	id := c.Param("id")
	span.SetAttributes(attribute.String("account.id", id))

	// Fetch account to check ownership before allowing update
	existing, getErr := h.GetUC.Execute(ctx, dto.GetInput{ID: id})
	if getErr != nil {
		HandleError(c, span, getErr)
		return
	}

	if !isAdminOrOwner(c, existing.UserID) {
		span.SetStatus(codes.Error, "forbidden")
		httpgin.SendError(c, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.UpdateInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid request body")
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ID = id

	res, execErr := h.UpdateUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Delete godoc
// @Summary      Delete an account
// @Description  Soft delete an account by ID
// @Tags         accounts
// @Produce      json
// @Param        id   path      string  true  "Account ID"
// @Success      204
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id} [delete]
func (h *AccountHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "AccountHandler.Delete")
	defer span.End()

	id := c.Param("id")
	span.SetAttributes(attribute.String("account.id", id))

	// Fetch account to check ownership before allowing delete
	existing, getErr := h.GetUC.Execute(ctx, dto.GetInput{ID: id})
	if getErr != nil {
		HandleError(c, span, getErr)
		return
	}

	if !isAdminOrOwner(c, existing.UserID) {
		span.SetStatus(codes.Error, "forbidden")
		httpgin.SendError(c, http.StatusForbidden, "forbidden")
		return
	}

	_, execErr := h.DeleteUC.Execute(ctx, dto.DeleteInput{ID: id})
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	c.Status(http.StatusNoContent)
}

package handler

import (
	"net/http"

	categoryuc "github.com/DenysonJ/financial-wallet/internal/usecases/category"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// CategoryHandler groups the HTTP handlers for the category domain.
type CategoryHandler struct {
	createUC *categoryuc.CreateUseCase
	listUC   *categoryuc.ListUseCase
	updateUC *categoryuc.UpdateUseCase
	deleteUC *categoryuc.DeleteUseCase
}

// NewCategoryHandler builds the handler with the injected use cases.
func NewCategoryHandler(
	createUC *categoryuc.CreateUseCase,
	listUC *categoryuc.ListUseCase,
	updateUC *categoryuc.UpdateUseCase,
	deleteUC *categoryuc.DeleteUseCase,
) *CategoryHandler {
	return &CategoryHandler{
		createUC: createUC,
		listUC:   listUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

// Create godoc
// @Summary      Create a custom category
// @Description  Creates a category in the authenticated user's scope. System defaults are seeded and not user-creatable.
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateInput true "Category info"
// @Success      201  {object}  dto.CreateOutput
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     jwt
// @Router       /categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "CategoryHandler.Create")
	defer span.End()

	var req dto.CreateInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, ok := getRequiredJWTUserID(c)
	if !ok {
		httpgin.SendError(c, http.StatusUnauthorized, "authentication required")
		return
	}
	req.UserID = userID

	span.SetAttributes(attribute.String("category.type", req.Type))

	res, execErr := h.createUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	span.SetAttributes(attribute.String("category.id", res.ID))
	httpgin.SendSuccess(c, http.StatusCreated, res)
}

// List godoc
// @Summary      List visible categories
// @Description  Returns the union of system defaults and the authenticated user's categories. Optional filters: type (credit|debit) and scope (system|user).
// @Tags         categories
// @Produce      json
// @Param        type   query     string  false  "Filter by type (credit|debit)"
// @Param        scope  query     string  false  "Filter by scope (system|user); omitted = both"
// @Success      200    {object}  dto.ListOutput
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     jwt
// @Router       /categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "CategoryHandler.List")
	defer span.End()

	var req dto.ListInput
	if bindErr := c.ShouldBindQuery(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	userID, ok := getRequiredJWTUserID(c)
	if !ok {
		httpgin.SendError(c, http.StatusUnauthorized, "authentication required")
		return
	}
	req.UserID = userID

	span.SetAttributes(
		attribute.String("filter.type", req.Type),
		attribute.String("filter.scope", req.Scope),
	)

	res, execErr := h.listUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Update godoc
// @Summary      Rename a custom category
// @Description  Renames a user-owned category. The type field is immutable; only the name is updated. System defaults are read-only.
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        id       path      string           true  "Category ID"
// @Param        request  body      dto.UpdateInput  true  "New name"
// @Success      200      {object}  dto.UpdateOutput
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     jwt
// @Router       /categories/{id} [patch]
func (h *CategoryHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "CategoryHandler.Update")
	defer span.End()

	var req dto.UpdateInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, ok := getRequiredJWTUserID(c)
	if !ok {
		httpgin.SendError(c, http.StatusUnauthorized, "authentication required")
		return
	}
	req.UserID = userID
	req.ID = c.Param("id")

	span.SetAttributes(attribute.String("category.id", req.ID))

	res, execErr := h.updateUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Delete godoc
// @Summary      Delete a custom category
// @Description  Deletes a user-owned category. Returns 409 if any statements still reference it; system defaults return 403.
// @Tags         categories
// @Produce      json
// @Param        id   path      string  true  "Category ID"
// @Success      204
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "CategoryHandler.Delete")
	defer span.End()

	userID, ok := getRequiredJWTUserID(c)
	if !ok {
		httpgin.SendError(c, http.StatusUnauthorized, "authentication required")
		return
	}

	id := c.Param("id")
	span.SetAttributes(attribute.String("category.id", id))

	_, execErr := h.deleteUC.Execute(ctx, dto.DeleteInput{
		UserID: userID,
		ID:     id,
	})
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	c.Status(http.StatusNoContent)
}

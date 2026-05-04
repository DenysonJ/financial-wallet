package handler

import (
	"net/http"

	taguc "github.com/DenysonJ/financial-wallet/internal/usecases/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// TagHandler groups the HTTP handlers for the tag domain.
type TagHandler struct {
	createUC *taguc.CreateUseCase
	listUC   *taguc.ListUseCase
	updateUC *taguc.UpdateUseCase
	deleteUC *taguc.DeleteUseCase
}

// NewTagHandler builds the handler with the injected use cases.
func NewTagHandler(
	createUC *taguc.CreateUseCase,
	listUC *taguc.ListUseCase,
	updateUC *taguc.UpdateUseCase,
	deleteUC *taguc.DeleteUseCase,
) *TagHandler {
	return &TagHandler{
		createUC: createUC,
		listUC:   listUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

// Create godoc
// @Summary      Create a custom tag
// @Description  Creates a tag in the authenticated user's scope.
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateInput true "Tag info"
// @Success      201  {object}  dto.CreateOutput
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /tags [post]
func (h *TagHandler) Create(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "TagHandler.Create")
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

	res, execErr := h.createUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	span.SetAttributes(attribute.String("tag.id", res.ID))
	httpgin.SendSuccess(c, http.StatusCreated, res)
}

// List godoc
// @Summary      List visible tags
// @Description  Returns the union of system default tags and the authenticated user's tags. Optional scope filter (system|user).
// @Tags         tags
// @Produce      json
// @Param        scope  query     string  false  "Filter by scope (system|user); omitted = both"
// @Success      200    {object}  dto.ListOutput
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /tags [get]
func (h *TagHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "TagHandler.List")
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

	span.SetAttributes(attribute.String("filter.scope", req.Scope))

	res, execErr := h.listUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Update godoc
// @Summary      Rename a custom tag
// @Description  Renames a user-owned tag. System defaults are read-only.
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        id       path      string           true  "Tag ID"
// @Param        request  body      dto.UpdateInput  true  "New name"
// @Success      200      {object}  dto.UpdateOutput
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /tags/{id} [patch]
func (h *TagHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "TagHandler.Update")
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

	span.SetAttributes(attribute.String("tag.id", req.ID))

	res, execErr := h.updateUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Delete godoc
// @Summary      Delete a custom tag
// @Description  Deletes a user-owned tag. Associations in statement_tags CASCADE; statements remain intact. System defaults return 403.
// @Tags         tags
// @Produce      json
// @Param        id   path      string  true  "Tag ID"
// @Success      204
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /tags/{id} [delete]
func (h *TagHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "TagHandler.Delete")
	defer span.End()

	userID, ok := getRequiredJWTUserID(c)
	if !ok {
		httpgin.SendError(c, http.StatusUnauthorized, "authentication required")
		return
	}

	id := c.Param("id")
	span.SetAttributes(attribute.String("tag.id", id))

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

package handler

import (
	"net/http"

	stmtuc "github.com/DenysonJ/financial-wallet/internal/usecases/statement"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// StatementMetadataHandler groups handlers that mutate metadata on existing
// statements. Split from StatementHandler because these routes (PATCH
// category, PUT tags) operate outside the append-only invariant — they never
// touch accounting fields.
type StatementMetadataHandler struct {
	updateCategoryUC *stmtuc.UpdateCategoryUseCase
	replaceTagsUC    *stmtuc.ReplaceTagsUseCase
}

// NewStatementMetadataHandler builds the handler.
func NewStatementMetadataHandler(
	updateCategoryUC *stmtuc.UpdateCategoryUseCase,
	replaceTagsUC *stmtuc.ReplaceTagsUseCase,
) *StatementMetadataHandler {
	return &StatementMetadataHandler{
		updateCategoryUC: updateCategoryUC,
		replaceTagsUC:    replaceTagsUC,
	}
}

// UpdateCategory godoc
// @Summary      Update or clear a statement's category
// @Description  Sets, swaps, or clears (with `null`) the category on an existing statement. The category type must match the statement type. This operation never modifies amount, type, or balance — it is purely a metadata mutation (REQ-11).
// @Tags         statements
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "Statement ID"
// @Param        request  body      dto.UpdateCategoryInput   true  "New category_id (UUID) or null to clear"
// @Success      200      {object}  dto.StatementOutput
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      422      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /statements/{id}/category [patch]
func (h *StatementMetadataHandler) UpdateCategory(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "StatementHandler.UpdateCategory")
	defer span.End()

	var req dto.UpdateCategoryInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	req.StatementID = c.Param("id")
	req.RequestingUserID = ownershipUserID(c)

	span.SetAttributes(attribute.String("statement.id", req.StatementID))

	res, execErr := h.updateCategoryUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// ReplaceTags godoc
// @Summary      Replace the tag set of a statement
// @Description  Replaces the entire tag set assigned to a statement (PUT semantics). Empty array clears all tags. This operation never modifies amount, type, or balance — it is purely a metadata mutation (REQ-10).
// @Tags         statements
// @Accept       json
// @Produce      json
// @Param        id       path      string                  true  "Statement ID"
// @Param        request  body      dto.ReplaceTagsInput    true  "Tag IDs (deduplicated; max 10)"
// @Success      200      {object}  dto.StatementOutput
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      422      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /statements/{id}/tags [put]
func (h *StatementMetadataHandler) ReplaceTags(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "StatementHandler.ReplaceTags")
	defer span.End()

	var req dto.ReplaceTagsInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	req.StatementID = c.Param("id")
	req.RequestingUserID = ownershipUserID(c)

	span.SetAttributes(
		attribute.String("statement.id", req.StatementID),
		attribute.Int("tags.count", len(req.TagIDs)),
	)

	res, execErr := h.replaceTagsUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

package handler

import (
	"net/http"

	roleuc "github.com/DenysonJ/financial-wallet/internal/usecases/role"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// RoleHandler agrupa todos os handlers relacionados a Role.
// Segue o padrão de injeção de dependência (UseCases injetados via struct).
type RoleHandler struct {
	createUC *roleuc.CreateUseCase
	listUC   *roleuc.ListUseCase
	deleteUC *roleuc.DeleteUseCase
	assignUC *roleuc.AssignRoleUseCase
	revokeUC *roleuc.RevokeRoleUseCase
}

// NewRoleHandler cria um novo RoleHandler com todos os use cases.
func NewRoleHandler(
	createUC *roleuc.CreateUseCase,
	listUC *roleuc.ListUseCase,
	deleteUC *roleuc.DeleteUseCase,
	assignUC *roleuc.AssignRoleUseCase,
	revokeUC *roleuc.RevokeRoleUseCase,
) *RoleHandler {
	return &RoleHandler{
		createUC: createUC,
		listUC:   listUC,
		deleteUC: deleteUC,
		assignUC: assignUC,
		revokeUC: revokeUC,
	}
}

// Create godoc
// @Summary      Create a new role
// @Description  Create a new role with the input payload
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateInput true "Role info"
// @Success      201  {object}  dto.CreateOutput
// @Failure      400  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "RoleHandler.Create")
	defer span.End()

	var req dto.CreateInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	span.SetAttributes(
		attribute.String("role.name", req.Name),
	)

	res, execErr := h.createUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	span.SetAttributes(attribute.String("role.id", res.ID))

	httpgin.SendSuccess(c, http.StatusCreated, res)
}

// List godoc
// @Summary      List roles
// @Description  Get a paginated list of roles
// @Tags         roles
// @Produce      json
// @Param        page   query     int     false  "Page number"
// @Param        limit  query     int     false  "Items per page"
// @Param        name   query     string  false  "Filter by name"
// @Success      200    {object}  dto.ListOutput
// @Failure      400   {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      429    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "RoleHandler.List")
	defer span.End()

	var req dto.ListInput
	if bindErr := c.ShouldBindQuery(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	span.SetAttributes(
		attribute.Int("filter.page", req.Page),
		attribute.Int("filter.limit", req.Limit),
	)

	res, execErr := h.listUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	span.SetAttributes(attribute.Int("result.total", res.Pagination.Total))
	httpgin.SendSuccessWithMeta(c, http.StatusOK, res.Data, res.Pagination, nil)
}

// Delete godoc
// @Summary      Delete a role
// @Description  Delete a role by ID
// @Tags         roles
// @Produce      json
// @Param        id   path      string  true  "Role ID"
// @Success      200  {object}  dto.DeleteOutput
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "RoleHandler.Delete")
	defer span.End()

	id := c.Param("id")
	span.SetAttributes(attribute.String("role.id", id))

	res, execErr := h.deleteUC.Execute(ctx, dto.DeleteInput{ID: id})
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// AssignRole godoc
// @Summary      Assign a role to a user
// @Description  Assign the specified role to a user
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "Role ID"
// @Param        request  body      dto.AssignRoleInput  true  "User to assign"
// @Success      204
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /roles/{id}/assign [post]
func (h *RoleHandler) AssignRole(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "RoleHandler.AssignRole")
	defer span.End()

	roleID := c.Param("id")
	span.SetAttributes(attribute.String("role.id", roleID))

	var req dto.AssignRoleInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	req.RoleID = roleID

	span.SetAttributes(attribute.String("user.id", req.UserID))

	execErr := h.assignUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokeRole godoc
// @Summary      Revoke a role from a user
// @Description  Revoke the specified role from a user
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id       path      string               true  "Role ID"
// @Param        request  body      dto.RevokeRoleInput   true  "User to revoke"
// @Success      204
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /roles/{id}/revoke [post]
func (h *RoleHandler) RevokeRole(c *gin.Context) {
	ctx, span := otel.Tracer(handlerTracer).Start(c.Request.Context(), "RoleHandler.RevokeRole")
	defer span.End()

	roleID := c.Param("id")
	span.SetAttributes(attribute.String("role.id", roleID))

	var req dto.RevokeRoleInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logutil.LogWarn(ctx, "bind error", "error", bindErr.Error())
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	req.RoleID = roleID

	span.SetAttributes(attribute.String("user.id", req.UserID))

	execErr := h.revokeUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, execErr)
		return
	}

	c.Status(http.StatusNoContent)
}

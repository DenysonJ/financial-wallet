package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	stmtuc "github.com/DenysonJ/financial-wallet/internal/usecases/statement"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// StatementHandler groups all handlers related to Statement.
type StatementHandler struct {
	createUC  *stmtuc.CreateUseCase
	reverseUC *stmtuc.ReverseUseCase
	getUC     *stmtuc.GetUseCase
	listUC    *stmtuc.ListUseCase
	importUC  *stmtuc.ImportUseCase
}

// NewStatementHandler creates a new StatementHandler with all use cases.
func NewStatementHandler(
	createUC *stmtuc.CreateUseCase,
	reverseUC *stmtuc.ReverseUseCase,
	getUC *stmtuc.GetUseCase,
	listUC *stmtuc.ListUseCase,
	importUC *stmtuc.ImportUseCase,
) *StatementHandler {
	return &StatementHandler{
		createUC:  createUC,
		reverseUC: reverseUC,
		getUC:     getUC,
		listUC:    listUC,
		importUC:  importUC,
	}
}

// Create godoc
// @Summary      Create a new statement
// @Description  Create a credit or debit statement for an account. Atomically updates the account balance.
// @Tags         statements
// @Accept       json
// @Produce      json
// @Param        id          path      string           true  "Account ID"
// @Param        request     body      dto.CreateInput  true  "Statement info (type: credit/debit, amount in cents)"
// @Success      201         {object}  dto.StatementOutput
// @Failure      400         {object}  ErrorResponse
// @Failure      401         {object}  ErrorResponse
// @Failure      404         {object}  ErrorResponse
// @Failure      409         {object}  ErrorResponse
// @Failure      422         {object}  ErrorResponse
// @Failure      429         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id}/statements [post]
func (h *StatementHandler) Create(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "StatementHandler.Create")
	defer span.End()

	accountID := c.Param("id")
	span.SetAttributes(attribute.String("account.id", accountID))

	var req dto.CreateInput
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid request body")
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	req.AccountID = accountID
	req.RequestingUserID = ownershipUserID(c)

	res, execErr := h.createUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	span.SetAttributes(attribute.String("statement.id", res.ID))
	httpgin.SendSuccess(c, http.StatusCreated, res)
}

// Reverse godoc
// @Summary      Reverse a statement
// @Description  Create a reversal statement with opposite type. A statement can only be reversed once.
// @Tags         statements
// @Accept       json
// @Produce      json
// @Param        id          path      string  true  "Account ID"
// @Param        statement_id  path    string  true  "Statement ID to reverse"
// @Param        request     body      dto.ReverseInput  false  "Optional reversal description"
// @Success      201         {object}  dto.StatementOutput
// @Failure      400         {object}  ErrorResponse
// @Failure      401         {object}  ErrorResponse
// @Failure      404         {object}  ErrorResponse
// @Failure      409         {object}  ErrorResponse
// @Failure      429         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id}/statements/{statement_id}/reverse [post]
func (h *StatementHandler) Reverse(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "StatementHandler.Reverse")
	defer span.End()

	accountID := c.Param("id")
	statementID := c.Param("statement_id")
	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("statement.id", statementID),
	)

	var req dto.ReverseInput
	// Body is optional (only description), so ignore bind errors for empty body
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil && !errors.Is(bindErr, io.EOF) {
		httpgin.SendError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	req.AccountID = accountID
	req.StatementID = statementID
	req.RequestingUserID = ownershipUserID(c)

	res, execErr := h.reverseUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	span.SetAttributes(attribute.String("reversal.id", res.ID))
	httpgin.SendSuccess(c, http.StatusCreated, res)
}

// List godoc
// @Summary      List statements by account
// @Description  Get a paginated list of statements for an account, with optional filters
// @Tags         statements
// @Produce      json
// @Param        id          path      string  true   "Account ID"
// @Param        type        query     string  false  "Filter by type (credit/debit)"
// @Param        date_from   query     string  false  "Filter from date (RFC3339)"
// @Param        date_to     query     string  false  "Filter to date (RFC3339)"
// @Param        page        query     int     false  "Page number"
// @Param        limit       query     int     false  "Items per page"
// @Success      200         {object}  dto.ListOutput
// @Failure      400         {object}  ErrorResponse
// @Failure      401         {object}  ErrorResponse
// @Failure      404         {object}  ErrorResponse
// @Failure      429         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id}/statements [get]
func (h *StatementHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "StatementHandler.List")
	defer span.End()

	accountID := c.Param("id")
	span.SetAttributes(attribute.String("account.id", accountID))

	var req dto.ListInput
	if bindErr := c.ShouldBindQuery(&req); bindErr != nil {
		span.SetStatus(codes.Error, "invalid query parameters")
		httpgin.SendError(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	req.AccountID = accountID
	req.RequestingUserID = ownershipUserID(c)

	res, execErr := h.listUC.Execute(ctx, req)
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	span.SetAttributes(attribute.Int("result.total", res.Pagination.Total))
	httpgin.SendSuccess(c, http.StatusOK, res)
}

// GetByID godoc
// @Summary      Get a statement by ID
// @Description  Get statement details by its unique ID within an account
// @Tags         statements
// @Produce      json
// @Param        id          path      string  true  "Account ID"
// @Param        statement_id  path    string  true  "Statement ID"
// @Success      200         {object}  dto.StatementOutput
// @Failure      404         {object}  ErrorResponse
// @Failure      429         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id}/statements/{statement_id} [get]
func (h *StatementHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "StatementHandler.GetByID")
	defer span.End()

	accountID := c.Param("id")
	statementID := c.Param("statement_id")
	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("statement.id", statementID),
	)

	res, execErr := h.getUC.Execute(ctx, dto.GetInput{
		ID:               statementID,
		AccountID:        accountID,
		RequestingUserID: ownershipUserID(c),
	})
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	httpgin.SendSuccess(c, http.StatusOK, res)
}

// Import godoc
// @Summary      Import statements from OFX file
// @Description  Parse an OFX bank statement file and batch-create statements. Duplicates are skipped via FITID.
// @Tags         statements
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      string  true  "Account ID"
// @Param        file  formData  file    true  "OFX file"
// @Success      200   {object}  dto.ImportOutput
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      422   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     ServiceName
// @Security     ServiceKey
// @Router       /accounts/{id}/statements/import [post]
func (h *StatementHandler) Import(c *gin.Context) {
	ctx, span := otel.Tracer("http-handler").Start(c.Request.Context(), "StatementHandler.Import")
	defer span.End()

	accountID := c.Param("id")
	span.SetAttributes(attribute.String("account.id", accountID))

	// Enforce 5MB file size limit
	const maxFileSize = 5 << 20 // 5MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	fileHeader, formErr := c.FormFile("file")
	if formErr != nil {
		span.SetStatus(codes.Error, "file upload failed")
		var maxBytesErr *http.MaxBytesError
		if errors.As(formErr, &maxBytesErr) {
			httpgin.SendError(c, http.StatusBadRequest, "file too large (max 5MB)")
			return
		}
		httpgin.SendError(c, http.StatusBadRequest, "file is required")
		return
	}

	file, openErr := fileHeader.Open()
	if openErr != nil {
		span.SetStatus(codes.Error, "failed to open file")
		httpgin.SendError(c, http.StatusBadRequest, "failed to read uploaded file")
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			span.SetStatus(codes.Error, "failed to close file")
		}
	}(file)

	res, execErr := h.importUC.Execute(ctx, dto.ImportOFXInput{
		AccountID:        accountID,
		RequestingUserID: ownershipUserID(c),
		FileContent:      file,
	})
	if execErr != nil {
		HandleError(c, span, execErr)
		return
	}

	span.SetAttributes(
		attribute.Int("import.created", res.Created),
		attribute.Int("import.skipped", res.Skipped),
	)
	httpgin.SendSuccess(c, http.StatusOK, res)
}

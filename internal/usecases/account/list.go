package account

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/account/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// ListUseCase implementa o caso de uso de listar accounts.
type ListUseCase struct {
	repo interfaces.Repository
}

// NewListUseCase cria uma nova instância do ListUseCase.
func NewListUseCase(repo interfaces.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

// Execute retorna uma lista paginada de accounts filtrada por user_id.
func (uc *ListUseCase) Execute(ctx context.Context, input dto.ListInput) (*dto.ListOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Account.List")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionList)

	// Validar UserID
	userID, parseErr := uservo.ParseID(input.UserID)
	if parseErr != nil {
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "account list failed: invalid user ID", "error", parseErr.Error())
		return nil, parseErr
	}

	span.SetAttributes(
		attribute.Int("filter.page", input.Page),
		attribute.Int("filter.limit", input.Limit),
		attribute.String("filter.user_id", input.UserID),
	)

	// Converter input para filtro de domínio
	filter := accountdomain.ListFilter{
		Page:       input.Page,
		Limit:      input.Limit,
		UserID:     userID,
		Name:       input.Name,
		Type:       input.Type,
		ActiveOnly: input.ActiveOnly,
	}

	// Buscar no repositório
	result, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		span.SetStatus(otelcodes.Error, listErr.Error())
		logutil.LogError(ctx, "account list failed: repository error", "error", listErr.Error())
		return nil, listErr
	}

	// Converter para DTOs de saída
	items := make([]dto.GetOutput, 0, len(result.Accounts))
	for _, a := range result.Accounts {
		items = append(items, dto.GetOutput{
			ID:          a.ID.String(),
			UserID:      a.UserID.String(),
			Name:        a.Name,
			Type:        a.Type.String(),
			Description: a.Description,
			Active:      a.Active,
			CreatedAt:   a.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
		})
	}

	totalPages := 0

	if result.Limit > 0 {
		totalPages = (result.Total + result.Limit - 1) / result.Limit
	}

	span.SetAttributes(attribute.Int("result.total", result.Total))
	logutil.LogInfo(ctx, "accounts listed", "total", result.Total, "page", result.Page)

	return &dto.ListOutput{
		Data: items,
		Pagination: dto.PaginationOutput{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

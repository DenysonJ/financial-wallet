package role

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/role/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// ListUseCase implementa o caso de uso de listar roles.
type ListUseCase struct {
	repo interfaces.Repository
}

// NewListUseCase cria uma nova instancia do ListUseCase.
func NewListUseCase(repo interfaces.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

// Execute retorna uma lista paginada de roles.
func (uc *ListUseCase) Execute(ctx context.Context, input dto.ListInput) (*dto.ListOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Role.List")
	defer span.End()

	ctx = injectLogContext(ctx, "list")

	span.SetAttributes(
		attribute.Int("filter.page", input.Page),
		attribute.Int("filter.limit", input.Limit),
	)

	// Converter input para filtro de dominio
	filter := roledomain.ListFilter{
		Page:  input.Page,
		Limit: input.Limit,
		Name:  input.Name,
	}

	// List operations return only infrastructure errors — no expected domain
	// sentinels apply, so FailSpan is used directly without IsExpected guard.
	result, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		telemetry.FailSpan(span, listErr, "role list failed")
		logutil.LogError(ctx, "role list failed: repository error", "error", listErr.Error())
		return nil, listErr
	}

	// Converter para DTOs de saida
	items := make([]dto.RoleOutput, 0, len(result.Roles))
	for _, r := range result.Roles {
		items = append(items, dto.RoleOutput{
			ID:          r.ID.String(),
			Name:        r.Name,
			Description: r.Description,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		})
	}

	totalPages := 0
	if result.Limit > 0 {
		totalPages = (result.Total + result.Limit - 1) / result.Limit
	}

	span.SetAttributes(attribute.Int("result.total", result.Total))
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "roles listed", "total", result.Total, "page", result.Page)

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

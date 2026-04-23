package user

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// ListUseCase implementa o caso de uso de listar users.
type ListUseCase struct {
	repo interfaces.Repository
}

// NewListUseCase cria uma nova instância do ListUseCase.
func NewListUseCase(repo interfaces.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

// Execute retorna uma lista paginada de users.
func (uc *ListUseCase) Execute(ctx context.Context, input dto.ListInput) (*dto.ListOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.User.List")
	defer span.End()

	ctx = injectLogContext(ctx, "list")

	span.SetAttributes(
		attribute.Int("filter.page", input.Page),
		attribute.Int("filter.limit", input.Limit),
	)

	// Converter input para filtro de domínio
	filter := userdomain.ListFilter{
		Page:       input.Page,
		Limit:      input.Limit,
		Name:       input.Name,
		Email:      input.Email,
		ActiveOnly: input.ActiveOnly,
	}

	// List operations return only infrastructure errors — no expected domain
	// sentinels apply, so FailSpan is used directly without IsExpected guard.
	result, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		telemetry.FailSpan(span, listErr, "user list failed")
		logutil.LogError(ctx, "user list failed: repository error", "error", listErr.Error())
		return nil, listErr
	}

	// Converter para DTOs de saída
	items := make([]dto.GetOutput, 0, len(result.Users))
	for _, e := range result.Users {
		items = append(items, dto.GetOutput{
			ID:        e.ID.String(),
			Name:      e.Name,
			Email:     e.Email.String(),
			Active:    e.Active,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
			UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
		})
	}

	totalPages := 0
	if result.Limit > 0 {
		totalPages = (result.Total + result.Limit - 1) / result.Limit
	}

	span.SetAttributes(attribute.Int("result.total", result.Total))
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "users listed", "total", result.Total, "page", result.Page)

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

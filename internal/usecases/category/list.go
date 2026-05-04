package category

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// ListUseCase lists categories visible to the user (defaults + own).
type ListUseCase struct {
	repo interfaces.Repository
}

// NewListUseCase builds the use case.
func NewListUseCase(repo interfaces.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

// Execute applies optional type and scope filters and returns the serialized result.
func (uc *ListUseCase) Execute(ctx context.Context, input dto.ListInput) (*dto.ListOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Category.List")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionList)

	userID, parseErr := pkgvo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		return nil, parseErr
	}

	filter := categorydomain.ListFilter{UserID: userID}

	if input.Type != "" {
		ct, typeErr := categoryvo.NewCategoryType(input.Type)
		if typeErr != nil {
			telemetry.WarnSpan(span, attribute.String("app.result", "invalid_type"))
			return nil, typeErr
		}
		filter.Type = &ct
	}

	switch input.Scope {
	case "system":
		filter.Scope = categorydomain.ScopeSystem
	case "user":
		filter.Scope = categorydomain.ScopeUser
	default:
		filter.Scope = categorydomain.ScopeAll
	}

	span.SetAttributes(
		attribute.String("filter.type", input.Type),
		attribute.String("filter.scope", string(filter.Scope)),
	)

	categories, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		telemetry.FailSpan(span, listErr, "category list failed")
		logutil.LogError(ctx, "category list failed", "error", listErr.Error())
		return nil, listErr
	}

	items := make([]dto.CategoryOutput, 0, len(categories))
	for _, c := range categories {
		items = append(items, dto.FromDomain(c))
	}

	span.SetAttributes(attribute.Int("result.count", len(items)))
	telemetry.OkSpan(span)

	return &dto.ListOutput{Data: items}, nil
}

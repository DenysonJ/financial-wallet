package tag

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// ListUseCase lists tags visible to the user (defaults + own).
type ListUseCase struct {
	repo interfaces.Repository
}

// NewListUseCase builds the use case.
func NewListUseCase(repo interfaces.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

// Execute applies the optional scope filter and returns the serialized result.
func (uc *ListUseCase) Execute(ctx context.Context, input dto.ListInput) (*dto.ListOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Tag.List")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionList)

	userID, parseErr := pkgvo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		return nil, parseErr
	}

	filter := tagdomain.ListFilter{UserID: userID}

	switch input.Scope {
	case "system":
		filter.Scope = tagdomain.ScopeSystem
	case "user":
		filter.Scope = tagdomain.ScopeUser
	default:
		filter.Scope = tagdomain.ScopeAll
	}

	span.SetAttributes(attribute.String("filter.scope", string(filter.Scope)))

	tags, listErr := uc.repo.List(ctx, filter)
	if listErr != nil {
		telemetry.FailSpan(span, listErr, "tag list failed")
		logutil.LogError(ctx, "tag list failed", "error", listErr.Error())
		return nil, listErr
	}

	items := make([]dto.TagOutput, 0, len(tags))
	for _, t := range tags {
		items = append(items, dto.FromDomain(t))
	}

	span.SetAttributes(attribute.Int("result.count", len(items)))
	telemetry.OkSpan(span)

	return &dto.ListOutput{Data: items}, nil
}

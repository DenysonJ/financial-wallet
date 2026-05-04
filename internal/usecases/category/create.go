package category

import (
	"context"
	"strings"

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

// CreateUseCase creates a user-scoped custom category.
type CreateUseCase struct {
	repo interfaces.Repository
}

// NewCreateUseCase builds the use case.
func NewCreateUseCase(repo interfaces.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

// Execute validates input, builds the entity, and persists it.
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.CreateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Category.Create")
	defer span.End()

	ctx = injectLogContext(ctx, "create")

	userID, parseErr := pkgvo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		return nil, parseErr
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_name"))
		return nil, categorydomain.ErrCategoryInvalidName
	}

	categoryType, typeErr := categoryvo.NewCategoryType(input.Type)
	if typeErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_type"))
		return nil, typeErr
	}

	c := categorydomain.NewCategory(userID, name, categoryType)

	if createErr := uc.repo.Create(ctx, c); createErr != nil {
		telemetry.ClassifyError(ctx, span, createErr, "domain_error", "category creation failed")
		return nil, createErr
	}

	span.SetAttributes(
		attribute.String("category.id", c.ID.String()),
		attribute.String("category.type", c.Type.String()),
	)
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "category created", "category.id", c.ID.String(), "category.type", c.Type.String())

	return &dto.CreateOutput{CategoryOutput: dto.FromDomain(c)}, nil
}

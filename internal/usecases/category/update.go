package category

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// UpdateUseCase renames a user-owned category. Type is immutable.
type UpdateUseCase struct {
	repo interfaces.Repository
}

// NewUpdateUseCase builds the use case.
func NewUpdateUseCase(repo interfaces.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

// Execute validates ownership/visibility, rejects system defaults, and persists the new name.
func (uc *UpdateUseCase) Execute(ctx context.Context, input dto.UpdateInput) (*dto.UpdateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Category.Update")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionUpdate)

	userID, userErr := pkgvo.ParseID(input.UserID)
	if userErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		return nil, userErr
	}

	id, idErr := pkgvo.ParseID(input.ID)
	if idErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		return nil, idErr
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_name"))
		return nil, categorydomain.ErrCategoryInvalidName
	}

	span.SetAttributes(attribute.String("category.id", input.ID))

	// Load without a visibility filter so we can distinguish system defaults
	// (ReadOnly) from cross-user (NotFound) before calling Update.
	c, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "domain_error", "category update failed")
		return nil, findErr
	}

	// System default → ReadOnly (403).
	if c.IsSystem() {
		telemetry.WarnSpan(span, attribute.String("app.result", "read_only"))
		return nil, categorydomain.ErrCategoryReadOnly
	}

	// Cross-user → NotFound (404, no existence oracle).
	if c.UserID == nil || *c.UserID != userID {
		telemetry.WarnSpan(span, attribute.String("app.result", "not_owner"))
		return nil, categorydomain.ErrCategoryNotFound
	}

	if renameErr := c.Rename(name); renameErr != nil {
		telemetry.ClassifyError(ctx, span, renameErr, "domain_error", "category update failed")
		return nil, renameErr
	}

	if updateErr := uc.repo.Update(ctx, c); updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "category update failed")
		return nil, updateErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "category updated", "category.id", c.ID.String())

	return &dto.UpdateOutput{CategoryOutput: dto.FromDomain(c)}, nil
}

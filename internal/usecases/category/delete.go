package category

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/category/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// DeleteUseCase removes a user-owned custom category. System defaults are
// immutable; categories in use return 409.
type DeleteUseCase struct {
	repo interfaces.Repository
}

// NewDeleteUseCase builds the use case.
func NewDeleteUseCase(repo interfaces.Repository) *DeleteUseCase {
	return &DeleteUseCase{repo: repo}
}

// Execute validates ownership, checks usage, and deletes.
func (uc *DeleteUseCase) Execute(ctx context.Context, input dto.DeleteInput) (*dto.DeleteOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Category.Delete")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionDelete)

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

	span.SetAttributes(attribute.String("category.id", input.ID))

	c, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "domain_error", "category delete failed")
		return nil, findErr
	}

	if c.IsSystem() {
		telemetry.WarnSpan(span, attribute.String("app.result", "read_only"))
		return nil, categorydomain.ErrCategoryReadOnly
	}

	if c.UserID == nil || *c.UserID != userID {
		telemetry.WarnSpan(span, attribute.String("app.result", "not_owner"))
		return nil, categorydomain.ErrCategoryNotFound
	}

	count, countErr := uc.repo.CountStatementsUsing(ctx, id)
	if countErr != nil {
		telemetry.FailSpan(span, countErr, "category delete failed: count statements")
		return nil, countErr
	}
	if count > 0 {
		telemetry.WarnSpan(span,
			attribute.String("app.result", "in_use"),
			attribute.Int("category.statements", count),
		)
		return nil, categorydomain.ErrCategoryInUse
	}

	if deleteErr := uc.repo.Delete(ctx, id); deleteErr != nil {
		telemetry.ClassifyError(ctx, span, deleteErr, "domain_error", "category delete failed")
		return nil, deleteErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "category deleted", "category.id", input.ID)

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

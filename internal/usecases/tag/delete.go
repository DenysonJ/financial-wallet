package tag

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// DeleteUseCase removes a user-owned custom tag.
//
// Unlike category, a tag in use CAN be deleted — `statement_tags` CASCADE
// drops the associations automatically. Statements remain intact.
type DeleteUseCase struct {
	repo interfaces.Repository
}

// NewDeleteUseCase builds the use case.
func NewDeleteUseCase(repo interfaces.Repository) *DeleteUseCase {
	return &DeleteUseCase{repo: repo}
}

// Execute validates ownership and deletes.
func (uc *DeleteUseCase) Execute(ctx context.Context, input dto.DeleteInput) (*dto.DeleteOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Tag.Delete")
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

	span.SetAttributes(attribute.String("tag.id", input.ID))

	t, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "domain_error", "tag delete failed")
		return nil, findErr
	}

	if t.IsSystem() {
		telemetry.WarnSpan(span, attribute.String("app.result", "read_only"))
		return nil, tagdomain.ErrTagReadOnly
	}

	if t.UserID == nil || *t.UserID != userID {
		telemetry.WarnSpan(span, attribute.String("app.result", "not_owner"))
		return nil, tagdomain.ErrTagNotFound
	}

	if deleteErr := uc.repo.Delete(ctx, id); deleteErr != nil {
		telemetry.ClassifyError(ctx, span, deleteErr, "domain_error", "tag delete failed")
		return nil, deleteErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "tag deleted", "tag.id", input.ID)

	return &dto.DeleteOutput{
		ID:        input.ID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

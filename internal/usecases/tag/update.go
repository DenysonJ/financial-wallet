package tag

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/tag/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// UpdateUseCase renames a user-owned tag.
type UpdateUseCase struct {
	repo interfaces.Repository
}

// NewUpdateUseCase builds the use case.
func NewUpdateUseCase(repo interfaces.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

// Execute validates ownership/visibility, rejects system defaults, and persists the new name.
func (uc *UpdateUseCase) Execute(ctx context.Context, input dto.UpdateInput) (*dto.UpdateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Tag.Update")
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
		return nil, tagdomain.ErrTagInvalidName
	}

	span.SetAttributes(attribute.String("tag.id", input.ID))

	t, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "domain_error", "tag update failed")
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

	if renameErr := t.Rename(name); renameErr != nil {
		telemetry.ClassifyError(ctx, span, renameErr, "domain_error", "tag update failed")
		return nil, renameErr
	}

	if updateErr := uc.repo.Update(ctx, t); updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "tag update failed")
		return nil, updateErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "tag updated", "tag.id", t.ID.String())

	return &dto.UpdateOutput{TagOutput: dto.FromDomain(t)}, nil
}

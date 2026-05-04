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

// CreateUseCase creates a user-scoped custom tag.
type CreateUseCase struct {
	repo interfaces.Repository
}

// NewCreateUseCase builds the use case.
func NewCreateUseCase(repo interfaces.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

// Execute validates input, builds the entity, and persists it.
func (uc *CreateUseCase) Execute(ctx context.Context, input dto.CreateInput) (*dto.CreateOutput, error) {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.Tag.Create")
	defer span.End()

	ctx = injectLogContext(ctx, logutil.ActionCreate)

	userID, parseErr := pkgvo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_user_id"))
		return nil, parseErr
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_name"))
		return nil, tagdomain.ErrTagInvalidName
	}

	t := tagdomain.NewTag(userID, name)

	if createErr := uc.repo.Create(ctx, t); createErr != nil {
		telemetry.ClassifyError(ctx, span, createErr, "domain_error", "tag creation failed")
		return nil, createErr
	}

	span.SetAttributes(attribute.String("tag.id", t.ID.String()))
	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "tag created", "tag.id", t.ID.String())

	return &dto.CreateOutput{TagOutput: dto.FromDomain(t)}, nil
}

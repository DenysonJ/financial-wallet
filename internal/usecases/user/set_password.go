package user

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// SetPasswordUseCase implementa o caso de uso de cadastro de senha.
type SetPasswordUseCase struct {
	repo       interfaces.Repository
	bcryptCost int
}

// NewSetPasswordUseCase cria uma nova instância do SetPasswordUseCase.
func NewSetPasswordUseCase(repo interfaces.Repository) *SetPasswordUseCase {
	return &SetPasswordUseCase{
		repo:       repo,
		bcryptCost: vo.DefaultBcryptCost,
	}
}

// WithBcryptCost sets a custom bcrypt cost (builder pattern).
func (uc *SetPasswordUseCase) WithBcryptCost(cost int) *SetPasswordUseCase {
	uc.bcryptCost = cost
	return uc
}

// Execute cadastra a senha de um usuário que ainda não possui senha.
//
// Fluxo:
//  1. Validar ID e buscar usuário
//  2. Verificar se já possui senha (ErrPasswordAlreadySet)
//  3. Validar que senha e confirmação coincidem
//  4. Criar hash bcrypt via Value Object
//  5. Persistir hash no banco
func (uc *SetPasswordUseCase) Execute(ctx context.Context, input dto.SetPasswordInput) error {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.User.SetPassword")
	defer span.End()

	ctx = injectLogContext(ctx, "set_password")

	id, parseErr := vo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		logutil.LogWarn(ctx, "set password failed: invalid ID", "error", parseErr.Error())
		return parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.UserID))

	e, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "set password failed")
		return findErr
	}

	if e.PasswordHash != "" {
		telemetry.WarnSpan(span, attribute.String("app.result", "password_already_set"))
		logutil.LogWarn(ctx, "set password failed: password already set")
		return userdomain.ErrPasswordAlreadySet
	}

	if input.Password != input.PasswordConfirmation {
		telemetry.WarnSpan(span, attribute.String("app.result", "password_mismatch"))
		logutil.LogWarn(ctx, "set password failed: password mismatch")
		return userdomain.ErrPasswordMismatch
	}

	passwordVO, hashErr := vo.NewPassword(input.Password, uc.bcryptCost)
	if hashErr != nil {
		telemetry.ClassifyError(ctx, span, hashErr, "invalid_password", "set password failed")
		return hashErr
	}

	updateErr := uc.repo.UpdatePassword(ctx, id, passwordVO.String())
	if updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "set password failed")
		return updateErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "password set", "user.id", input.UserID)

	return nil
}

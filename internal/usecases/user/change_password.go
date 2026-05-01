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

// ChangePasswordUseCase implementa o caso de uso de alteração de senha.
type ChangePasswordUseCase struct {
	repo       interfaces.Repository
	bcryptCost int
}

// NewChangePasswordUseCase cria uma nova instância do ChangePasswordUseCase.
func NewChangePasswordUseCase(repo interfaces.Repository) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		repo:       repo,
		bcryptCost: vo.DefaultBcryptCost,
	}
}

// WithBcryptCost sets a custom bcrypt cost (builder pattern).
func (uc *ChangePasswordUseCase) WithBcryptCost(cost int) *ChangePasswordUseCase {
	uc.bcryptCost = cost
	return uc
}

// Execute altera a senha de um usuário autenticado.
//
// Fluxo:
//  1. Validar ID e buscar usuário
//  2. Verificar senha atual contra o hash armazenado
//  3. Validar que nova senha e confirmação coincidem
//  4. Criar novo hash bcrypt via Value Object
//  5. Persistir novo hash no banco
func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input dto.ChangePasswordInput) error {
	ctx, span := otel.Tracer(TracerKey).Start(ctx, "UseCase.User.ChangePassword")
	defer span.End()

	ctx = injectLogContext(ctx, "change_password")

	id, parseErr := vo.ParseID(input.UserID)
	if parseErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_id"))
		logutil.LogWarn(ctx, "change password failed: invalid ID", "error", parseErr.Error())
		return parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.UserID))

	e, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		telemetry.ClassifyError(ctx, span, findErr, "not_found", "change password failed")
		return findErr
	}

	checkErr := vo.CheckPassword(e.PasswordHash, input.CurrentPassword)
	if checkErr != nil {
		telemetry.WarnSpan(span, attribute.String("app.result", "invalid_credentials"))
		// audit=password_change_failed lets ops alert on brute-force attempts
		logutil.LogWarn(ctx, "change password failed: invalid current password",
			"audit", "password_change_failed",
			"reason", "invalid_credentials",
			"user.id", input.UserID)
		return userdomain.ErrInvalidCredentials
	}

	if input.NewPassword != input.NewPasswordConfirmation {
		telemetry.WarnSpan(span, attribute.String("app.result", "password_mismatch"))
		logutil.LogWarn(ctx, "change password failed: password mismatch",
			"audit", "password_change_failed",
			"reason", "password_mismatch",
			"user.id", input.UserID)
		return userdomain.ErrPasswordMismatch
	}

	passwordVO, hashErr := vo.NewPassword(input.NewPassword, uc.bcryptCost)
	if hashErr != nil {
		telemetry.ClassifyError(ctx, span, hashErr, "invalid_password", "change password failed")
		return hashErr
	}

	updateErr := uc.repo.UpdatePassword(ctx, id, passwordVO.String())
	if updateErr != nil {
		telemetry.ClassifyError(ctx, span, updateErr, "domain_error", "change password failed")
		return updateErr
	}

	telemetry.OkSpan(span)
	logutil.LogInfo(ctx, "password changed",
		"audit", "password_change",
		"user.id", input.UserID)

	return nil
}

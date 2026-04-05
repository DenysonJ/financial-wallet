package user

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
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
		span.SetStatus(otelcodes.Error, parseErr.Error())
		logutil.LogWarn(ctx, "change password failed: invalid ID", "error", parseErr.Error())
		return parseErr
	}

	span.SetAttributes(attribute.String("user.id", input.UserID))

	e, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		span.SetStatus(otelcodes.Error, findErr.Error())
		logutil.LogWarn(ctx, "change password failed", "error", findErr.Error())
		return findErr
	}

	checkErr := vo.CheckPassword(e.PasswordHash, input.CurrentPassword)
	if checkErr != nil {
		span.SetStatus(otelcodes.Error, userdomain.ErrInvalidCredentials.Error())
		logutil.LogWarn(ctx, "change password failed: invalid current password")
		return userdomain.ErrInvalidCredentials
	}

	if input.NewPassword != input.NewPasswordConfirmation {
		span.SetStatus(otelcodes.Error, userdomain.ErrPasswordMismatch.Error())
		logutil.LogWarn(ctx, "change password failed: password mismatch")
		return userdomain.ErrPasswordMismatch
	}

	passwordVO, hashErr := vo.NewPassword(input.NewPassword, uc.bcryptCost)
	if hashErr != nil {
		span.SetStatus(otelcodes.Error, hashErr.Error())
		logutil.LogWarn(ctx, "change password failed: validation error", "error", hashErr.Error())
		return hashErr
	}

	updateErr := uc.repo.UpdatePassword(ctx, id, passwordVO.String())
	if updateErr != nil {
		span.SetStatus(otelcodes.Error, updateErr.Error())
		logutil.LogError(ctx, "change password failed: repository error", "error", updateErr.Error())
		return updateErr
	}

	logutil.LogInfo(ctx, "password changed", "user.id", input.UserID)

	return nil
}

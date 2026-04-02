package user

import (
	"context"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
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
	id, parseErr := vo.ParseID(input.UserID)
	if parseErr != nil {
		return parseErr
	}

	e, findErr := uc.repo.FindByID(ctx, id)
	if findErr != nil {
		return findErr
	}

	checkErr := vo.CheckPassword(e.PasswordHash, input.CurrentPassword)
	if checkErr != nil {
		return userdomain.ErrInvalidCredentials
	}

	if input.NewPassword != input.NewPasswordConfirmation {
		return userdomain.ErrPasswordMismatch
	}

	passwordVO, hashErr := vo.NewPassword(input.NewPassword, uc.bcryptCost)
	if hashErr != nil {
		return hashErr
	}

	updateErr := uc.repo.UpdatePassword(ctx, id, passwordVO.String())
	if updateErr != nil {
		return updateErr
	}

	return nil
}

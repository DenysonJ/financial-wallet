package user

import (
	"context"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/dto"
	"github.com/DenysonJ/financial-wallet/internal/usecases/user/interfaces"
)

// SetPasswordUseCase implementa o caso de uso de cadastro de senha.
type SetPasswordUseCase struct {
	Repo       interfaces.Repository
	BcryptCost int
}

// NewSetPasswordUseCase cria uma nova instância do SetPasswordUseCase.
func NewSetPasswordUseCase(repo interfaces.Repository) *SetPasswordUseCase {
	return &SetPasswordUseCase{
		Repo:       repo,
		BcryptCost: vo.DefaultBcryptCost,
	}
}

// WithBcryptCost sets a custom bcrypt cost (builder pattern).
func (uc *SetPasswordUseCase) WithBcryptCost(cost int) *SetPasswordUseCase {
	uc.BcryptCost = cost
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
	id, parseErr := vo.ParseID(input.UserID)
	if parseErr != nil {
		return parseErr
	}

	e, findErr := uc.Repo.FindByID(ctx, id)
	if findErr != nil {
		return findErr
	}

	if e.PasswordHash != "" {
		return userdomain.ErrPasswordAlreadySet
	}

	if input.Password != input.PasswordConfirmation {
		return userdomain.ErrPasswordMismatch
	}

	passwordVO, hashErr := vo.NewPassword(input.Password, uc.BcryptCost)
	if hashErr != nil {
		return hashErr
	}

	updateErr := uc.Repo.UpdatePassword(ctx, id, passwordVO.String())
	if updateErr != nil {
		return updateErr
	}

	return nil
}

package interfaces

import (
	"context"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Repository define o contrato para persistência de Account.
type Repository interface {
	// Create persiste uma nova Account.
	Create(ctx context.Context, a *accountdomain.Account) error

	// FindByID busca uma Account pelo ID.
	// Retorna ErrAccountNotFound se não encontrar.
	FindByID(ctx context.Context, id uservo.ID) (*accountdomain.Account, error)

	// List retorna uma lista paginada de Accounts com filtros.
	List(ctx context.Context, filter accountdomain.ListFilter) (*accountdomain.ListResult, error)

	// Update atualiza uma Account existente.
	// Retorna ErrAccountNotFound se o ID não existir.
	Update(ctx context.Context, a *accountdomain.Account) error

	// Delete realiza soft delete (active=false) de uma Account.
	// Retorna ErrAccountNotFound se o ID não existir.
	Delete(ctx context.Context, id uservo.ID) error
}

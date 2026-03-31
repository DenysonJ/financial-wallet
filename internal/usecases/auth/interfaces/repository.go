package interfaces

import (
	"context"

	userdomain "github.com/DenysonJ/financial-wallet/internal/domain/user"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
)

// UserRepository define o contrato para busca de usuários no contexto de autenticação.
type UserRepository interface {
	// FindByEmail busca um User pelo email.
	// Retorna ErrUserNotFound se não encontrar.
	FindByEmail(ctx context.Context, email vo.Email) (*userdomain.User, error)
}

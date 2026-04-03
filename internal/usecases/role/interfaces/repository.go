package interfaces

import (
	"context"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
)

// Repository define o CONTRATO para persistencia de Role.
//
// Esta e uma INTERFACE definida na camada de USE CASES e IMPLEMENTADA na INFRASTRUCTURE.
// Isso e a essencia da inversao de dependencia (Dependency Inversion Principle).
//
// Beneficios:
//   - Use cases nao sabem nada sobre banco de dados
//   - Facil trocar implementacao (Postgres -> MySQL)
//   - Facil criar mocks para testes
type Repository interface {
	// Create persiste uma nova Role no banco de dados.
	Create(ctx context.Context, r *roledomain.Role) error

	// List retorna uma lista paginada de Roles com filtros opcionais.
	List(ctx context.Context, filter roledomain.ListFilter) (*roledomain.ListResult, error)

	// Delete remove uma Role pelo ID.
	// Retorna ErrRoleNotFound se o ID nao existir.
	Delete(ctx context.Context, id vo.ID) error

	// FindByName busca uma Role pelo nome.
	// Retorna ErrRoleNotFound se nao encontrar.
	FindByName(ctx context.Context, name string) (*roledomain.Role, error)

	// FindByID busca uma Role pelo ID.
	// Retorna ErrRoleNotFound se nao encontrar.
	FindByID(ctx context.Context, id vo.ID) (*roledomain.Role, error)

	// AssignRole atribui uma role a um usuário.
	// Retorna ErrRoleAlreadyAssigned se a associação já existir.
	AssignRole(ctx context.Context, userID vo.ID, roleID vo.ID) error

	// RevokeRole revoga uma role de um usuário.
	// Retorna ErrRoleNotAssigned se a associação não existir.
	RevokeRole(ctx context.Context, userID vo.ID, roleID vo.ID) error

	// GetUserPermissions retorna a lista de permission names de um usuário.
	GetUserPermissions(ctx context.Context, userID vo.ID) ([]string, error)

	// GetUserRoles retorna a lista de role names de um usuário.
	GetUserRoles(ctx context.Context, userID vo.ID) ([]string, error)
}

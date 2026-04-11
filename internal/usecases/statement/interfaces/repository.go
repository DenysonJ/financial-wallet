package interfaces

import (
	"context"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Repository defines the contract for Statement persistence.
// Create is transactional: it inserts the statement and updates the account balance atomically.
type Repository interface {
	// Create persists a new Statement and updates the account balance in a single transaction.
	Create(ctx context.Context, stmt *stmtdomain.Statement, accountID vo.ID) error

	// FindByID returns a Statement by its ID.
	// Returns ErrStatementNotFound if not found.
	FindByID(ctx context.Context, id vo.ID) (*stmtdomain.Statement, error)

	// List returns a paginated list of Statements matching the filter.
	List(ctx context.Context, filter stmtdomain.ListFilter) (*stmtdomain.ListResult, error)

	// HasReversal checks if the given statement has already been reversed.
	HasReversal(ctx context.Context, statementID vo.ID) (bool, error)
}

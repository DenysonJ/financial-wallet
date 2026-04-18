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
	// Returns the balance after the statement was applied.
	Create(ctx context.Context, stmt *stmtdomain.Statement, accountID vo.ID) (int64, error)

	// FindByID returns a Statement by its ID.
	// Returns ErrStatementNotFound if not found.
	FindByID(ctx context.Context, id vo.ID) (*stmtdomain.Statement, error)

	// List returns a paginated list of Statements matching the filter.
	List(ctx context.Context, filter stmtdomain.ListFilter) (*stmtdomain.ListResult, error)

	// HasReversal checks if the given statement has already been reversed.
	HasReversal(ctx context.Context, statementID vo.ID) (bool, error)

	// CreateBatch persists multiple Statements and updates the account balance atomically.
	// Returns the final balance after all statements were applied.
	CreateBatch(ctx context.Context, stmts []*stmtdomain.Statement, accountID vo.ID) (int64, error)

	// FindExternalIDs returns which of the given external IDs already exist for the account.
	//
	// Implementations MUST query the writer (primary), not the reader replica.
	// OFX imports rely on this lookup to deduplicate FITIDs; if the replica
	// lag is non-zero, a just-imported FITID would be re-inserted on a retry
	// that lands between write and replication. Consistency beats read-load
	// distribution here.
	FindExternalIDs(ctx context.Context, accountID vo.ID, externalIDs []string) (map[string]bool, error)
}

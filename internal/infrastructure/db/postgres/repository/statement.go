package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
)

// statementDB is the database model for Statement.
type statementDB struct {
	ID           string    `db:"id"`
	AccountID    string    `db:"account_id"`
	Type         string    `db:"type"`
	Amount       int64     `db:"amount"`
	Description  string    `db:"description"`
	ReferenceID  *string   `db:"reference_id"`
	BalanceAfter int64     `db:"balance_after"`
	CreatedAt    time.Time `db:"created_at"`
}

func (s *statementDB) toStatement() (*stmtdomain.Statement, error) {
	id, parseErr := pkgvo.ParseID(s.ID)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing statement ID: %w", parseErr)
	}

	accountID, accountIDErr := pkgvo.ParseID(s.AccountID)
	if accountIDErr != nil {
		return nil, fmt.Errorf("parsing account ID: %w", accountIDErr)
	}

	stmt := &stmtdomain.Statement{
		ID:           id,
		AccountID:    accountID,
		Type:         stmtvo.ParseStatementType(s.Type),
		Amount:       stmtvo.ParseAmount(s.Amount),
		Description:  s.Description,
		BalanceAfter: s.BalanceAfter,
		CreatedAt:    s.CreatedAt,
	}

	if s.ReferenceID != nil {
		refID, refErr := pkgvo.ParseID(*s.ReferenceID)
		if refErr != nil {
			return nil, fmt.Errorf("parsing reference ID: %w", refErr)
		}
		stmt.ReferenceID = &refID
	}

	return stmt, nil
}

func fromDomainStatement(s *stmtdomain.Statement) statementDB {
	db := statementDB{
		ID:           s.ID.String(),
		AccountID:    s.AccountID.String(),
		Type:         s.Type.String(),
		Amount:       s.Amount.Int64(),
		Description:  s.Description,
		BalanceAfter: s.BalanceAfter,
		CreatedAt:    s.CreatedAt,
	}
	if s.ReferenceID != nil {
		ref := s.ReferenceID.String()
		db.ReferenceID = &ref
	}
	return db
}

// StatementRepository implements the Repository interface for Statement.
type StatementRepository struct {
	writer *sqlx.DB
	reader *sqlx.DB
}

// NewStatementRepository creates a new StatementRepository instance.
func NewStatementRepository(writer, reader *sqlx.DB) *StatementRepository {
	return &StatementRepository{writer: writer, reader: reader}
}

// Create inserts a new statement and atomically updates the account balance.
func (r *StatementRepository) Create(ctx context.Context, stmt *stmtdomain.Statement, accountID pkgvo.ID) error {
	tx, txErr := r.writer.BeginTxx(ctx, nil)
	if txErr != nil {
		return fmt.Errorf("beginning transaction: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock account row and get current balance
	var currentBalance int64
	lockErr := tx.GetContext(ctx, &currentBalance,
		"SELECT balance FROM accounts WHERE id = $1 FOR UPDATE",
		accountID.String(),
	)
	if lockErr != nil {
		if errors.Is(lockErr, sql.ErrNoRows) {
			return accountdomain.ErrAccountNotFound
		}
		return fmt.Errorf("locking account: %w", lockErr)
	}

	// Calculate new balance (can go negative — equivalent to owing money)
	amount := stmt.Amount.Int64()
	var newBalance int64
	if stmt.Type == stmtvo.TypeCredit {
		newBalance = currentBalance + amount
	} else {
		newBalance = currentBalance - amount
	}

	// Set balance after on the statement entity
	stmt.SetBalanceAfter(newBalance)

	// Insert statement
	dbModel := fromDomainStatement(stmt)
	insertQuery := `
		INSERT INTO statements (
			id, account_id, type, amount, description, reference_id, balance_after, created_at
		) VALUES (
			:id, :account_id, :type, :amount, :description, :reference_id, :balance_after, :created_at
		)
	`
	_, insertErr := tx.NamedExecContext(ctx, insertQuery, dbModel)
	if insertErr != nil {
		return fmt.Errorf("inserting statement: %w", insertErr)
	}

	// Update account balance
	_, updateErr := tx.ExecContext(ctx,
		"UPDATE accounts SET balance = $1, updated_at = $2 WHERE id = $3",
		newBalance, time.Now(), accountID.String(),
	)
	if updateErr != nil {
		return fmt.Errorf("updating account balance: %w", updateErr)
	}

	return tx.Commit()
}

// FindByID returns a Statement by its ID.
func (r *StatementRepository) FindByID(ctx context.Context, id pkgvo.ID) (*stmtdomain.Statement, error) {
	query := `
		SELECT id, account_id, type, amount, description, reference_id, balance_after, created_at
		FROM statements
		WHERE id = $1
	`

	var dbModel statementDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, stmtdomain.ErrStatementNotFound
		}
		return nil, selectErr
	}

	return dbModel.toStatement()
}

// List returns a paginated list of Statements matching the filter.
func (r *StatementRepository) List(ctx context.Context, filter stmtdomain.ListFilter) (*stmtdomain.ListResult, error) {
	filter.Normalize()

	conditions := []string{"account_id = :account_id"}
	args := map[string]interface{}{"account_id": filter.AccountID.String()}

	if filter.Type != nil {
		conditions = append(conditions, "type = :type")
		args["type"] = filter.Type.String()
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, "created_at >= :date_from")
		args["date_from"] = *filter.DateFrom
	}
	if filter.DateTo != nil {
		conditions = append(conditions, "created_at <= :date_to")
		args["date_to"] = *filter.DateTo
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Read-only transaction for consistent pagination
	tx, txErr := r.reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if txErr != nil {
		return nil, fmt.Errorf("beginning read transaction: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM statements %s", whereClause)
	countQuery, countArgs, namedErr := sqlx.Named(countQuery, args)
	if namedErr != nil {
		return nil, namedErr
	}
	countQuery = tx.Rebind(countQuery)

	var total int
	countErr := tx.GetContext(ctx, &total, countQuery, countArgs...)
	if countErr != nil {
		return nil, countErr
	}

	// Paginated data query
	args["limit"] = filter.Limit
	args["offset"] = filter.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT id, account_id, type, amount, description, reference_id, balance_after, created_at
		FROM statements
		%s
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`, whereClause)

	dataQuery, dataArgs, dataNamedErr := sqlx.Named(dataQuery, args)
	if dataNamedErr != nil {
		return nil, dataNamedErr
	}
	dataQuery = tx.Rebind(dataQuery)

	var dbModels []statementDB
	selectErr := tx.SelectContext(ctx, &dbModels, dataQuery, dataArgs...)
	if selectErr != nil {
		return nil, selectErr
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		return nil, fmt.Errorf("committing read transaction: %w", commitErr)
	}

	// Convert to domain statements
	statements := make([]*stmtdomain.Statement, 0, len(dbModels))
	for i := range dbModels {
		stmt, convertErr := dbModels[i].toStatement()
		if convertErr != nil {
			return nil, convertErr
		}
		statements = append(statements, stmt)
	}

	return &stmtdomain.ListResult{
		Statements: statements,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
	}, nil
}

// HasReversal checks if the given statement has already been reversed.
func (r *StatementRepository) HasReversal(ctx context.Context, statementID pkgvo.ID) (bool, error) {
	var exists bool
	queryErr := r.reader.GetContext(ctx, &exists,
		"SELECT EXISTS(SELECT 1 FROM statements WHERE reference_id = $1)",
		statementID.String(),
	)
	return exists, queryErr
}

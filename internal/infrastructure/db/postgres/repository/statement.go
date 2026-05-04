package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// statementDB is the database model for Statement, including hydrated
// category + tags (LEFT JOIN + json_agg subselect).
type statementDB struct {
	ID           string         `db:"id"`
	AccountID    string         `db:"account_id"`
	Type         string         `db:"type"`
	Amount       int64          `db:"amount"`
	Description  string         `db:"description"`
	ReferenceID  *string        `db:"reference_id"`
	ExternalID   *string        `db:"external_id"`
	BalanceAfter int64          `db:"balance_after"`
	PostedAt     time.Time      `db:"posted_at"`
	CreatedAt    time.Time      `db:"created_at"`
	CategoryID   sql.NullString `db:"category_id"`
	// Hydrated columns from LEFT JOIN categories (NULL when CategoryID is NULL).
	CategoryName sql.NullString `db:"category_name"`
	CategoryType sql.NullString `db:"category_type"`
	// json_agg result: '[{"id":"...","name":"..."}]' — never NULL (COALESCE in query).
	TagsJSON []byte `db:"tags_json"`
}

// tagJSONRow models a single object inside tags_json.
type tagJSONRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
		ExternalID:   s.ExternalID,
		BalanceAfter: s.BalanceAfter,
		PostedAt:     s.PostedAt,
		CreatedAt:    s.CreatedAt,
		Tags:         []stmtdomain.TagRef{}, // never nil
	}

	if s.ReferenceID != nil {
		refID, refErr := pkgvo.ParseID(*s.ReferenceID)
		if refErr != nil {
			return nil, fmt.Errorf("parsing reference ID: %w", refErr)
		}
		stmt.ReferenceID = &refID
	}

	if s.CategoryID.Valid {
		catID, catIDErr := pkgvo.ParseID(s.CategoryID.String)
		if catIDErr != nil {
			return nil, fmt.Errorf("parsing category ID: %w", catIDErr)
		}
		stmt.CategoryID = &catID
		if s.CategoryName.Valid && s.CategoryType.Valid {
			stmt.Category = &stmtdomain.CategoryRef{
				ID:   catID,
				Name: s.CategoryName.String,
				Type: stmtvo.ParseStatementType(s.CategoryType.String),
			}
		}
	}

	if len(s.TagsJSON) > 0 && string(s.TagsJSON) != "[]" {
		var rows []tagJSONRow
		if jsonErr := json.Unmarshal(s.TagsJSON, &rows); jsonErr != nil {
			return nil, fmt.Errorf("parsing tags_json: %w", jsonErr)
		}
		refs := make([]stmtdomain.TagRef, 0, len(rows))
		for _, row := range rows {
			tagID, tagIDErr := pkgvo.ParseID(row.ID)
			if tagIDErr != nil {
				return nil, fmt.Errorf("parsing tag ID: %w", tagIDErr)
			}
			refs = append(refs, stmtdomain.TagRef{ID: tagID, Name: row.Name})
		}
		stmt.Tags = refs
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
		ExternalID:   s.ExternalID,
		BalanceAfter: s.BalanceAfter,
		PostedAt:     s.PostedAt,
		CreatedAt:    s.CreatedAt,
	}
	if s.ReferenceID != nil {
		ref := s.ReferenceID.String()
		db.ReferenceID = &ref
	}
	if s.CategoryID != nil {
		db.CategoryID = sql.NullString{String: s.CategoryID.String(), Valid: true}
	}
	return db
}

// statementSelectColumns is the canonical SELECT projection for hydrated reads.
// Centralized so FindByID and List stay aligned.
const statementSelectColumns = `
	s.id, s.account_id, s.type, s.amount, s.description, s.reference_id, s.external_id,
	s.balance_after, s.posted_at, s.created_at, s.category_id,
	c.name AS category_name, c.type AS category_type,
	COALESCE(
		(SELECT json_agg(json_build_object('id', t.id, 'name', t.name) ORDER BY LOWER(t.name))
		 FROM statement_tags st JOIN tags t ON t.id = st.tag_id
		 WHERE st.statement_id = s.id),
		'[]'::json
	) AS tags_json
`

// statementSelectFrom is the canonical FROM + LEFT JOIN clause for hydrated reads.
const statementSelectFrom = `
	FROM statements s
	LEFT JOIN categories c ON c.id = s.category_id
`

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
func (r *StatementRepository) Create(ctx context.Context, stmt *stmtdomain.Statement, accountID pkgvo.ID) (int64, error) {
	tx, txErr := r.writer.BeginTxx(ctx, nil)
	if txErr != nil {
		return 0, fmt.Errorf("beginning transaction: %w", txErr)
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
			return 0, accountdomain.ErrAccountNotFound
		}
		return 0, fmt.Errorf("locking account: %w", lockErr)
	}

	// Calculate new balance (can go negative — equivalent to owing money)
	amount := stmt.Amount.Int64()
	var newBalance int64
	if stmt.Type == stmtvo.TypeCredit {
		newBalance = currentBalance + amount
	} else {
		newBalance = currentBalance - amount
	}

	// Insert statement with calculated balance
	dbModel := fromDomainStatement(stmt)
	dbModel.BalanceAfter = newBalance
	insertQuery := `
		INSERT INTO statements (
			id, account_id, type, amount, description, reference_id, external_id, balance_after, posted_at, created_at, category_id
		) VALUES (
			:id, :account_id, :type, :amount, :description, :reference_id, :external_id, :balance_after, :posted_at, :created_at, :category_id
		)
	`
	_, insertErr := tx.NamedExecContext(ctx, insertQuery, dbModel)
	if insertErr != nil {
		return 0, fmt.Errorf("inserting statement: %w", insertErr)
	}

	// Insert tag associations (if any)
	if tagsErr := insertStatementTagsTx(ctx, tx, stmt.ID, stmt.Tags); tagsErr != nil {
		return 0, fmt.Errorf("inserting statement tags: %w", tagsErr)
	}

	// Update account balance
	_, updateErr := tx.ExecContext(ctx,
		"UPDATE accounts SET balance = $1, updated_at = NOW() WHERE id = $2",
		newBalance, accountID.String(),
	)
	if updateErr != nil {
		return 0, fmt.Errorf("updating account balance: %w", updateErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return 0, commitErr
	}
	return newBalance, nil
}

// insertStatementTagsTx inserts (statement_id, tag_id) rows for the given
// statement inside an open transaction. No-op when refs is empty.
func insertStatementTagsTx(ctx context.Context, tx *sqlx.Tx, statementID pkgvo.ID, refs []stmtdomain.TagRef) error {
	if len(refs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(refs))
	args := make([]interface{}, 0, len(refs)*2)
	for i, ref := range refs {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, statementID.String(), ref.ID.String())
	}
	query := "INSERT INTO statement_tags (statement_id, tag_id) VALUES " + strings.Join(placeholders, ",")
	_, execErr := tx.ExecContext(ctx, query, args...)
	return execErr
}

// FindByID returns a Statement by its ID, hydrated with category + tags.
func (r *StatementRepository) FindByID(ctx context.Context, id pkgvo.ID) (*stmtdomain.Statement, error) {
	query := "SELECT " + statementSelectColumns + statementSelectFrom + " WHERE s.id = $1"

	var dbModel statementDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, stmtdomain.ErrStatementNotFound
		}
		return nil, fmt.Errorf("finding statement by ID: %w", selectErr)
	}

	return dbModel.toStatement()
}

// List returns a paginated list of Statements matching the filter, hydrated
// with category + tags via JOIN + json_agg.
func (r *StatementRepository) List(ctx context.Context, filter stmtdomain.ListFilter) (*stmtdomain.ListResult, error) {
	filter.Normalize()

	conditions := []string{"s.account_id = :account_id"}
	args := map[string]interface{}{"account_id": filter.AccountID.String()}

	if filter.Type != nil {
		conditions = append(conditions, "s.type = :type")
		args["type"] = filter.Type.String()
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, "s.posted_at >= :date_from")
		args["date_from"] = *filter.DateFrom
	}
	if filter.DateTo != nil {
		conditions = append(conditions, "s.posted_at <= :date_to")
		args["date_to"] = *filter.DateTo
	}
	if filter.CategoryID != nil {
		conditions = append(conditions, "s.category_id = :category_id")
		args["category_id"] = filter.CategoryID.String()
	}
	if len(filter.TagIDs) > 0 {
		// Match statements that have ANY of the given tag IDs assigned.
		tagStrs := make([]string, len(filter.TagIDs))
		for i, id := range filter.TagIDs {
			tagStrs[i] = id.String()
		}
		conditions = append(conditions, "EXISTS (SELECT 1 FROM statement_tags stf WHERE stf.statement_id = s.id AND stf.tag_id = ANY(:tag_ids))")
		args["tag_ids"] = pq.Array(tagStrs)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Read-only transaction for consistent pagination
	tx, txErr := r.reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if txErr != nil {
		return nil, fmt.Errorf("beginning read transaction: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM statements s %s", whereClause)
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

	// Paginated data query — use keyset (cursor) when available, fall back to OFFSET
	args["limit"] = filter.Limit

	var dataQuery string
	if filter.UseCursor() {
		conditions = append(conditions, "(s.posted_at, s.id) < (:cursor_posted_at, :cursor_id)")
		args["cursor_posted_at"] = *filter.CursorPostedAt
		args["cursor_id"] = filter.CursorID.String()
		cursorWhere := "WHERE " + strings.Join(conditions, " AND ")
		dataQuery = "SELECT " + statementSelectColumns + statementSelectFrom + cursorWhere + `
			ORDER BY s.posted_at DESC, s.id DESC
			LIMIT :limit`
	} else {
		args["offset"] = filter.Offset()
		dataQuery = "SELECT " + statementSelectColumns + statementSelectFrom + whereClause + `
			ORDER BY s.posted_at DESC, s.id DESC
			LIMIT :limit OFFSET :offset`
	}

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

	// Build next cursor from last row (if we got a full page, there may be more)
	var nextCursor string
	if len(statements) == filter.Limit && len(statements) > 0 {
		last := statements[len(statements)-1]
		raw := last.PostedAt.Format(time.RFC3339Nano) + "|" + last.ID.String()
		nextCursor = base64.URLEncoding.EncodeToString([]byte(raw))
	}

	return &stmtdomain.ListResult{
		Statements: statements,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		NextCursor: nextCursor,
	}, nil
}

// HasReversal checks if the given statement has already been reversed.
// Runs on the writer to avoid replication-lag races where two concurrent
// reverse requests both observe no existing reversal.
func (r *StatementRepository) HasReversal(ctx context.Context, statementID pkgvo.ID) (bool, error) {
	var exists bool
	queryErr := r.writer.GetContext(ctx, &exists,
		"SELECT EXISTS(SELECT 1 FROM statements WHERE reference_id = $1)",
		statementID.String(),
	)
	if queryErr != nil {
		return false, fmt.Errorf("checking reversal: %w", queryErr)
	}
	return exists, nil
}

// CreateBatch inserts multiple statements and atomically updates the account balance.
func (r *StatementRepository) CreateBatch(ctx context.Context, stmts []*stmtdomain.Statement, accountID pkgvo.ID) (int64, error) {
	if len(stmts) == 0 {
		return 0, nil
	}

	tx, txErr := r.writer.BeginTxx(ctx, nil)
	if txErr != nil {
		return 0, fmt.Errorf("beginning transaction: %w", txErr)
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
			return 0, accountdomain.ErrAccountNotFound
		}
		return 0, fmt.Errorf("locking account: %w", lockErr)
	}

	// Compute running balance and prepare all DB models
	const colCount = 11 // 10 + category_id
	runningBalance := currentBalance
	placeholders := make([]string, 0, len(stmts))
	allArgs := make([]interface{}, 0, len(stmts)*colCount)

	for i, stmt := range stmts {
		amount := stmt.Amount.Int64()
		if stmt.Type == stmtvo.TypeCredit {
			runningBalance += amount
		} else {
			runningBalance -= amount
		}

		// Compute balance_after directly on the DB model.
		dbModel := fromDomainStatement(stmt)
		dbModel.BalanceAfter = runningBalance

		var categoryIDArg interface{}
		if dbModel.CategoryID.Valid {
			categoryIDArg = dbModel.CategoryID.String
		} else {
			categoryIDArg = nil
		}

		base := i * colCount
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5,
			base+6, base+7, base+8, base+9, base+10, base+11,
		))
		allArgs = append(allArgs,
			dbModel.ID, dbModel.AccountID, dbModel.Type, dbModel.Amount,
			dbModel.Description, dbModel.ReferenceID, dbModel.ExternalID,
			dbModel.BalanceAfter, dbModel.PostedAt, dbModel.CreatedAt,
			categoryIDArg,
		)
	}

	insertQuery := "INSERT INTO statements (id, account_id, type, amount, description, reference_id, external_id, balance_after, posted_at, created_at, category_id) VALUES " +
		strings.Join(placeholders, ",")

	_, insertErr := tx.ExecContext(ctx, insertQuery, allArgs...)
	if insertErr != nil {
		return 0, fmt.Errorf("inserting statement batch: %w", insertErr)
	}

	// Insert tag associations for each statement (best-effort batch — small N).
	for _, stmt := range stmts {
		if tagsErr := insertStatementTagsTx(ctx, tx, stmt.ID, stmt.Tags); tagsErr != nil {
			return 0, fmt.Errorf("inserting statement_tags: %w", tagsErr)
		}
	}

	// Update account balance with final balance
	_, updateErr := tx.ExecContext(ctx,
		"UPDATE accounts SET balance = $1, updated_at = NOW() WHERE id = $2",
		runningBalance, accountID.String(),
	)
	if updateErr != nil {
		return 0, fmt.Errorf("updating account balance: %w", updateErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return 0, commitErr
	}
	return runningBalance, nil
}

// FindExternalIDs returns which of the given external IDs already exist for the account.
func (r *StatementRepository) FindExternalIDs(ctx context.Context, accountID pkgvo.ID, externalIDs []string) (map[string]bool, error) {
	if len(externalIDs) == 0 {
		return make(map[string]bool), nil
	}

	query := `SELECT external_id FROM statements WHERE account_id = $1 AND external_id = ANY($2)`

	var found []string
	// Use writer to avoid replication lag race during import dedup checks
	selectErr := r.writer.SelectContext(ctx, &found, query, accountID.String(), pq.Array(externalIDs))
	if selectErr != nil {
		return nil, fmt.Errorf("finding external IDs: %w", selectErr)
	}

	result := make(map[string]bool, len(found))
	for _, id := range found {
		result[id] = true
	}
	return result, nil
}

// UpdateCategory sets (or clears with nil) the category of a statement.
//
// This query updates ONLY `category_id` and `updated_at`.
// Accounting fields (amount, type, balance_after) are excluded
// from the SET clause — account balance and downstream statements stay intact.
func (r *StatementRepository) UpdateCategory(ctx context.Context, statementID pkgvo.ID, categoryID *pkgvo.ID) error {
	var (
		query string
		args  []interface{}
	)
	if categoryID != nil {
		query = `UPDATE statements SET category_id = $1 WHERE id = $2`
		args = []interface{}{categoryID.String(), statementID.String()}
	} else {
		query = `UPDATE statements SET category_id = NULL WHERE id = $1`
		args = []interface{}{statementID.String()}
	}

	result, execErr := r.writer.ExecContext(ctx, query, args...)
	if execErr != nil {
		if isForeignKeyViolation(execErr) {
			// Race: category was deleted between use case visibility check and UPDATE.
			return stmtdomain.ErrStatementNotFound
		}
		return fmt.Errorf("updating statement category: %w", execErr)
	}

	rows, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}
	if rows == 0 {
		return stmtdomain.ErrStatementNotFound
	}
	return nil
}

// ReplaceTags replaces the entire tag set of a statement.
// DELETE existing rows + INSERT new set, in a single transaction.
// Empty tagIDs is valid — clears all tags.
//
// Does NOT touch the parent statement row — accounting invariants preserved.
func (r *StatementRepository) ReplaceTags(ctx context.Context, statementID pkgvo.ID, tagIDs []pkgvo.ID) error {
	tx, txErr := r.writer.BeginTxx(ctx, nil)
	if txErr != nil {
		return fmt.Errorf("beginning transaction: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	// Verify the statement exists (defense — caller should have checked).
	var exists bool
	checkErr := tx.GetContext(ctx, &exists,
		"SELECT EXISTS(SELECT 1 FROM statements WHERE id = $1)",
		statementID.String(),
	)
	if checkErr != nil {
		return fmt.Errorf("checking statement existence: %w", checkErr)
	}
	if !exists {
		return stmtdomain.ErrStatementNotFound
	}

	if _, delErr := tx.ExecContext(ctx,
		"DELETE FROM statement_tags WHERE statement_id = $1",
		statementID.String(),
	); delErr != nil {
		return fmt.Errorf("deleting statement_tags: %w", delErr)
	}

	if len(tagIDs) > 0 {
		refs := make([]stmtdomain.TagRef, 0, len(tagIDs))
		for _, id := range tagIDs {
			refs = append(refs, stmtdomain.TagRef{ID: id})
		}
		if insertErr := insertStatementTagsTx(ctx, tx, statementID, refs); insertErr != nil {
			return fmt.Errorf("inserting statement_tags: %w", insertErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return commitErr
	}
	return nil
}

// CountByCategory returns how many statements reference the given category.
// Used by the category Delete use case to surface ErrCategoryInUse before
// issuing the DELETE.
func (r *StatementRepository) CountByCategory(ctx context.Context, categoryID pkgvo.ID) (int, error) {
	var count int
	queryErr := r.reader.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM statements WHERE category_id = $1",
		categoryID.String(),
	)
	if queryErr != nil {
		return 0, fmt.Errorf("counting statements by category: %w", queryErr)
	}
	return count, nil
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
)

// accountDB é o modelo de banco de dados para Account.
type accountDB struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	Name        string    `db:"name"`
	Type        string    `db:"type"`
	Description string    `db:"description"`
	Active      bool      `db:"active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (a *accountDB) toAccount() (*accountdomain.Account, error) {
	id, parseErr := uservo.ParseID(a.ID)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing account ID: %w", parseErr)
	}

	userID, userIDErr := uservo.ParseID(a.UserID)
	if userIDErr != nil {
		return nil, fmt.Errorf("parsing user ID: %w", userIDErr)
	}

	return &accountdomain.Account{
		ID:          id,
		UserID:      userID,
		Name:        a.Name,
		Type:        accountvo.ParseAccountType(a.Type),
		Description: a.Description,
		Active:      a.Active,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}, nil
}

func fromDomainAccount(a *accountdomain.Account) accountDB {
	return accountDB{
		ID:          a.ID.String(),
		UserID:      a.UserID.String(),
		Name:        a.Name,
		Type:        a.Type.String(),
		Description: a.Description,
		Active:      a.Active,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// AccountRepository implementa a interface Repository para Account.
type AccountRepository struct {
	writer *sqlx.DB
	reader *sqlx.DB
}

// NewAccountRepository cria uma nova instância do repositório.
func NewAccountRepository(writer, reader *sqlx.DB) *AccountRepository {
	return &AccountRepository{writer: writer, reader: reader}
}

func (r *AccountRepository) Create(ctx context.Context, a *accountdomain.Account) error {
	query := `
		INSERT INTO accounts (
			id, user_id, name, type, description, active, created_at, updated_at
		) VALUES (
			:id, :user_id, :name, :type, :description, :active, :created_at, :updated_at
		)
	`

	dbModel := fromDomainAccount(a)
	_, execErr := r.writer.NamedExecContext(ctx, query, dbModel)
	return execErr
}

func (r *AccountRepository) FindByID(ctx context.Context, id uservo.ID) (*accountdomain.Account, error) {
	query := `
		SELECT id, user_id, name, type, description, active, created_at, updated_at
		FROM accounts
		WHERE id = $1 AND active = true
	`

	var dbModel accountDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, accountdomain.ErrAccountNotFound
		}
		return nil, selectErr
	}

	return dbModel.toAccount()
}

func (r *AccountRepository) List(ctx context.Context, filter accountdomain.ListFilter) (*accountdomain.ListResult, error) {
	filter.Normalize()

	// user_id is mandatory — reject queries without it to prevent cross-user data leak
	if filter.UserID.String() == "" {
		return nil, fmt.Errorf("list accounts: user_id is required")
	}

	// Build dynamic query with filters
	conditions := []string{"user_id = :user_id"}
	args := map[string]interface{}{"user_id": filter.UserID.String()}

	if filter.ActiveOnly {
		conditions = append(conditions, "active = true")
	}
	if filter.Name != "" {
		conditions = append(conditions, "name ILIKE :name")
		args["name"] = "%" + filter.Name + "%"
	}
	if filter.Type != "" {
		conditions = append(conditions, "type = :type")
		args["type"] = filter.Type
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Read-only transaction for consistent pagination
	tx, txErr := r.reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if txErr != nil {
		return nil, fmt.Errorf("beginning read transaction: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM accounts %s", whereClause)

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
		SELECT id, user_id, name, type, description, active, created_at, updated_at
		FROM accounts
		%s
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`, whereClause)

	dataQuery, dataArgs, dataNamedErr := sqlx.Named(dataQuery, args)
	if dataNamedErr != nil {
		return nil, dataNamedErr
	}
	dataQuery = tx.Rebind(dataQuery)

	var dbModels []accountDB
	selectErr := tx.SelectContext(ctx, &dbModels, dataQuery, dataArgs...)
	if selectErr != nil {
		return nil, selectErr
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		return nil, fmt.Errorf("committing read transaction: %w", commitErr)
	}

	// Convert to domain accounts
	accounts := make([]*accountdomain.Account, 0, len(dbModels))
	for i := range dbModels {
		acc, convertErr := dbModels[i].toAccount()
		if convertErr != nil {
			return nil, convertErr
		}
		accounts = append(accounts, acc)
	}

	return &accountdomain.ListResult{
		Accounts: accounts,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
	}, nil
}

func (r *AccountRepository) Update(ctx context.Context, a *accountdomain.Account) error {
	query := `
		UPDATE accounts SET
			name = :name,
			description = :description,
			active = :active,
			updated_at = :updated_at
		WHERE id = :id
	`

	dbModel := fromDomainAccount(a)
	result, execErr := r.writer.NamedExecContext(ctx, query, dbModel)
	if execErr != nil {
		return execErr
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}

	if rowsAffected == 0 {
		return accountdomain.ErrAccountNotFound
	}

	return nil
}

func (r *AccountRepository) Delete(ctx context.Context, id uservo.ID) error {
	query := `
		UPDATE accounts SET
			active = false,
			updated_at = $1
		WHERE id = $2 AND active = true
	`

	result, execErr := r.writer.ExecContext(ctx, query, time.Now(), id.String())
	if execErr != nil {
		return execErr
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}

	if rowsAffected == 0 {
		return accountdomain.ErrAccountNotFound
	}

	return nil
}

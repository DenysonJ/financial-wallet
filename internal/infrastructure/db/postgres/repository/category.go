package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
)

// categoryDB is the DB-shape data model for Category.
//
// UserID is sql.NullString because the schema allows NULL (system categories).
type categoryDB struct {
	ID        string         `db:"id"`
	UserID    sql.NullString `db:"user_id"`
	Name      string         `db:"name"`
	Type      string         `db:"type"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func (c *categoryDB) toCategory() (*categorydomain.Category, error) {
	id, parseErr := pkgvo.ParseID(c.ID)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing category ID: %w", parseErr)
	}

	var userID *pkgvo.ID
	if c.UserID.Valid {
		uid, uidErr := pkgvo.ParseID(c.UserID.String)
		if uidErr != nil {
			return nil, fmt.Errorf("parsing category user_id: %w", uidErr)
		}
		userID = &uid
	}

	return &categorydomain.Category{
		ID:        id,
		UserID:    userID,
		Name:      c.Name,
		Type:      categoryvo.ParseCategoryType(c.Type),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func fromDomainCategory(c *categorydomain.Category) categoryDB {
	userID := sql.NullString{}
	if c.UserID != nil {
		userID = sql.NullString{String: c.UserID.String(), Valid: true}
	}
	return categoryDB{
		ID:        c.ID.String(),
		UserID:    userID,
		Name:      c.Name,
		Type:      c.Type.String(),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// CategoryRepository implementa interfaces.Repository para Category.
type CategoryRepository struct {
	writer *sqlx.DB
	reader *sqlx.DB
}

// NewCategoryRepository builds a CategoryRepository.
func NewCategoryRepository(writer, reader *sqlx.DB) *CategoryRepository {
	return &CategoryRepository{writer: writer, reader: reader}
}

func (r *CategoryRepository) Create(ctx context.Context, c *categorydomain.Category) error {
	query := `
		INSERT INTO categories (
			id, user_id, name, type, created_at, updated_at
		) VALUES (
			:id, :user_id, :name, :type, :created_at, :updated_at
		)
	`

	dbModel := fromDomainCategory(c)
	_, execErr := r.writer.NamedExecContext(ctx, query, dbModel)
	if execErr != nil {
		if isUniqueViolation(execErr) {
			return categorydomain.ErrCategoryDuplicate
		}
		return execErr
	}
	return nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id pkgvo.ID) (*categorydomain.Category, error) {
	query := `
		SELECT id, user_id, name, type, created_at, updated_at
		FROM categories
		WHERE id = $1
	`

	var dbModel categoryDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, categorydomain.ErrCategoryNotFound
		}
		return nil, selectErr
	}
	return dbModel.toCategory()
}

func (r *CategoryRepository) FindVisible(ctx context.Context, id, userID pkgvo.ID) (*categorydomain.Category, error) {
	query := `
		SELECT id, user_id, name, type, created_at, updated_at
		FROM categories
		WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
	`

	var dbModel categoryDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String(), userID.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, categorydomain.ErrCategoryNotVisible
		}
		return nil, selectErr
	}
	return dbModel.toCategory()
}

func (r *CategoryRepository) List(ctx context.Context, filter categorydomain.ListFilter) ([]*categorydomain.Category, error) {
	var (
		query string
		args  []any
	)

	switch filter.Scope {
	case categorydomain.ScopeSystem:
		query = `
			SELECT id, user_id, name, type, created_at, updated_at
			FROM categories
			WHERE user_id IS NULL`
		args = []any{}
	case categorydomain.ScopeUser:
		query = `
			SELECT id, user_id, name, type, created_at, updated_at
			FROM categories
			WHERE user_id = $1`
		args = []any{filter.UserID.String()}
	default: // ScopeAll (zero-value)
		query = `
			SELECT id, user_id, name, type, created_at, updated_at
			FROM categories
			WHERE user_id = $1 OR user_id IS NULL`
		args = []any{filter.UserID.String()}
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, filter.Type.String())
	}

	// Defaults first (NULL user_id sorts before user IDs in ASC), then alpha.
	query += " ORDER BY user_id NULLS FIRST, LOWER(name) ASC"

	var dbModels []categoryDB
	selectErr := r.reader.SelectContext(ctx, &dbModels, query, args...)
	if selectErr != nil {
		return nil, selectErr
	}

	out := make([]*categorydomain.Category, 0, len(dbModels))
	for i := range dbModels {
		c, convErr := dbModels[i].toCategory()
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, c)
	}
	return out, nil
}

// Update mutates only Name and updated_at.
//
// The use case is responsible for validating that the category is not a
// system default (IsSystem) and that it belongs to the user before calling
// this method. The repository is intentionally dumb — it just runs the UPDATE.
func (r *CategoryRepository) Update(ctx context.Context, c *categorydomain.Category) error {
	query := `
		UPDATE categories
		SET name = $1, updated_at = $2
		WHERE id = $3
	`

	result, execErr := r.writer.ExecContext(ctx, query, c.Name, c.UpdatedAt, c.ID.String())
	if execErr != nil {
		if isUniqueViolation(execErr) {
			return categorydomain.ErrCategoryDuplicate
		}
		return execErr
	}

	rows, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}
	if rows == 0 {
		return categorydomain.ErrCategoryNotFound
	}
	return nil
}

// Delete removes a category by ID.
//
// If the FK ON DELETE RESTRICT is violated (a statement still references it),
// returns ErrCategoryInUse — guards against a race between CountStatementsUsing
// (called by the use case) and the DELETE.
func (r *CategoryRepository) Delete(ctx context.Context, id pkgvo.ID) error {
	query := `DELETE FROM categories WHERE id = $1`

	result, execErr := r.writer.ExecContext(ctx, query, id.String())
	if execErr != nil {
		if isForeignKeyViolation(execErr) {
			return categorydomain.ErrCategoryInUse
		}
		return execErr
	}

	rows, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}
	if rows == 0 {
		return categorydomain.ErrCategoryNotFound
	}
	return nil
}

// CountStatementsUsing returns how many statements reference the category.
func (r *CategoryRepository) CountStatementsUsing(ctx context.Context, id pkgvo.ID) (int, error) {
	query := `SELECT COUNT(*) FROM statements WHERE category_id = $1`

	var count int
	selectErr := r.reader.GetContext(ctx, &count, query, id.String())
	if selectErr != nil {
		return 0, selectErr
	}
	return count, nil
}

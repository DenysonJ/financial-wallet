package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// tagDB is the DB-shape data model for Tag.
type tagDB struct {
	ID        string         `db:"id"`
	UserID    sql.NullString `db:"user_id"`
	Name      string         `db:"name"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func (t *tagDB) toTag() (*tagdomain.Tag, error) {
	id, parseErr := pkgvo.ParseID(t.ID)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing tag ID: %w", parseErr)
	}

	var userID *pkgvo.ID
	if t.UserID.Valid {
		uid, uidErr := pkgvo.ParseID(t.UserID.String)
		if uidErr != nil {
			return nil, fmt.Errorf("parsing tag user_id: %w", uidErr)
		}
		userID = &uid
	}

	return &tagdomain.Tag{
		ID:        id,
		UserID:    userID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}, nil
}

func fromDomainTag(t *tagdomain.Tag) tagDB {
	userID := sql.NullString{}
	if t.UserID != nil {
		userID = sql.NullString{String: t.UserID.String(), Valid: true}
	}
	return tagDB{
		ID:        t.ID.String(),
		UserID:    userID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// TagRepository implementa interfaces.Repository para Tag.
type TagRepository struct {
	writer *sqlx.DB
	reader *sqlx.DB
}

// NewTagRepository builds a TagRepository.
func NewTagRepository(writer, reader *sqlx.DB) *TagRepository {
	return &TagRepository{writer: writer, reader: reader}
}

func (r *TagRepository) Create(ctx context.Context, t *tagdomain.Tag) error {
	query := `
		INSERT INTO tags (
			id, user_id, name, created_at, updated_at
		) VALUES (
			:id, :user_id, :name, :created_at, :updated_at
		)
	`

	dbModel := fromDomainTag(t)
	_, execErr := r.writer.NamedExecContext(ctx, query, dbModel)
	if execErr != nil {
		if isUniqueViolation(execErr) {
			return tagdomain.ErrTagDuplicate
		}
		return execErr
	}
	return nil
}

func (r *TagRepository) FindByID(ctx context.Context, id pkgvo.ID) (*tagdomain.Tag, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM tags
		WHERE id = $1
	`

	var dbModel tagDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, tagdomain.ErrTagNotFound
		}
		return nil, selectErr
	}
	return dbModel.toTag()
}

func (r *TagRepository) FindVisible(ctx context.Context, id, userID pkgvo.ID) (*tagdomain.Tag, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM tags
		WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
	`

	var dbModel tagDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String(), userID.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, tagdomain.ErrTagNotVisible
		}
		return nil, selectErr
	}
	return dbModel.toTag()
}

// FindManyVisible returns only the tags visible to the user (own or defaults).
// Missing tags and tags owned by other users are silently dropped — the caller
// compares `len(ids)` with `len(returned)` to detect invalid IDs.
func (r *TagRepository) FindManyVisible(ctx context.Context, ids []pkgvo.ID, userID pkgvo.ID) ([]*tagdomain.Tag, error) {
	if len(ids) == 0 {
		return []*tagdomain.Tag{}, nil
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = id.String()
	}

	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM tags
		WHERE id = ANY($1) AND (user_id = $2 OR user_id IS NULL)
	`

	var dbModels []tagDB
	selectErr := r.reader.SelectContext(ctx, &dbModels, query, pq.Array(idStrs), userID.String())
	if selectErr != nil {
		return nil, selectErr
	}

	out := make([]*tagdomain.Tag, 0, len(dbModels))
	for i := range dbModels {
		t, convErr := dbModels[i].toTag()
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TagRepository) List(ctx context.Context, filter tagdomain.ListFilter) ([]*tagdomain.Tag, error) {
	var (
		query string
		args  []any
	)

	switch filter.Scope {
	case tagdomain.ScopeSystem:
		query = `
			SELECT id, user_id, name, created_at, updated_at
			FROM tags
			WHERE user_id IS NULL`
		args = []any{}
	case tagdomain.ScopeUser:
		query = `
			SELECT id, user_id, name, created_at, updated_at
			FROM tags
			WHERE user_id = $1`
		args = []any{filter.UserID.String()}
	default: // ScopeAll
		query = `
			SELECT id, user_id, name, created_at, updated_at
			FROM tags
			WHERE user_id = $1 OR user_id IS NULL`
		args = []any{filter.UserID.String()}
	}

	query += " ORDER BY user_id NULLS FIRST, LOWER(name) ASC"

	var dbModels []tagDB
	selectErr := r.reader.SelectContext(ctx, &dbModels, query, args...)
	if selectErr != nil {
		return nil, selectErr
	}

	out := make([]*tagdomain.Tag, 0, len(dbModels))
	for i := range dbModels {
		tag, convErr := dbModels[i].toTag()
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, tag)
	}
	return out, nil
}

func (r *TagRepository) Update(ctx context.Context, t *tagdomain.Tag) error {
	query := `
		UPDATE tags
		SET name = $1, updated_at = $2
		WHERE id = $3
	`

	result, execErr := r.writer.ExecContext(ctx, query, t.Name, t.UpdatedAt, t.ID.String())
	if execErr != nil {
		if isUniqueViolation(execErr) {
			return tagdomain.ErrTagDuplicate
		}
		return execErr
	}

	rows, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}
	if rows == 0 {
		return tagdomain.ErrTagNotFound
	}
	return nil
}

func (r *TagRepository) Delete(ctx context.Context, id pkgvo.ID) error {
	query := `DELETE FROM tags WHERE id = $1`

	result, execErr := r.writer.ExecContext(ctx, query, id.String())
	if execErr != nil {
		return execErr
	}

	rows, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}
	if rows == 0 {
		return tagdomain.ErrTagNotFound
	}
	return nil
}

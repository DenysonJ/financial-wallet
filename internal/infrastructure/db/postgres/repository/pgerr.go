package repository

import (
	"errors"

	"github.com/lib/pq"
)

// Postgres SQLSTATE codes used across repositories.
const (
	// pgUniqueViolation — unique_violation (e.g., duplicate key for unique index).
	pgUniqueViolation = "23505"
	// pgForeignKeyViolation — foreign_key_violation (e.g., DELETE blocked by RESTRICT FK).
	pgForeignKeyViolation = "23503"
)

// isUniqueViolation reports whether err is a Postgres unique_violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return string(pgErr.Code) == pgUniqueViolation
	}
	return false
}

// isForeignKeyViolation reports whether err is a Postgres foreign_key_violation (23503).
func isForeignKeyViolation(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return string(pgErr.Code) == pgForeignKeyViolation
	}
	return false
}

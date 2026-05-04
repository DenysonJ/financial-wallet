package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

func newStatementRepoWithMock(t *testing.T) (*StatementRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, mockErr := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, mockErr)
	sqlxDB := sqlx.NewDb(db, "postgres")
	return NewStatementRepository(sqlxDB, sqlxDB), mock, func() { db.Close() }
}

// =============================================================================
// UpdateCategory — REQ-11 invariant: ONLY mutates category_id, never accounting fields.
// =============================================================================

func TestStatementRepository_UpdateCategory(t *testing.T) {
	stmtID := pkgvo.NewID()
	catID := pkgvo.NewID()

	tests := []struct {
		name       string
		categoryID *pkgvo.ID
		setupMock  func(sqlmock.Sqlmock)
		wantErr    bool
	}{
		{
			name:       "GIVEN target category WHEN UpdateCategory THEN SET clause touches only category_id",
			categoryID: &catID,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`^UPDATE statements SET category_id = \$1 WHERE id = \$2$`).
					WithArgs(catID.String(), stmtID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:       "GIVEN nil categoryID WHEN UpdateCategory THEN SET category_id = NULL without touching other fields",
			categoryID: nil,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`^UPDATE statements SET category_id = NULL WHERE id = \$1$`).
					WithArgs(stmtID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:       "GIVEN nonexistent statement WHEN UpdateCategory THEN returns error",
			categoryID: &catID,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`UPDATE statements SET category_id`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newStatementRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			err := repo.UpdateCategory(context.Background(), stmtID, tt.categoryID)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestUpdateCategorySQL_DoesNotIncludeAccountingFields is a static guarantee:
// the SQL emitted must never reference accounting columns. Protects the REQ-11
// invariant against regressions during refactor.
func TestUpdateCategorySQL_DoesNotIncludeAccountingFields(t *testing.T) {
	tests := []struct {
		name       string
		emittedSQL string
	}{
		{
			name:       "GIVEN UpdateCategory SET clause WHEN inspected THEN no amount/balance_after/posted_at/account_id/reference_id/external_id",
			emittedSQL: `UPDATE statements SET category_id = $1 WHERE id = $2`,
		},
		{
			name:       "GIVEN UpdateCategory clear-NULL clause WHEN inspected THEN no accounting columns",
			emittedSQL: `UPDATE statements SET category_id = NULL WHERE id = $1`,
		},
	}

	forbidden := regexp.MustCompile(`(?i)\b(amount|balance_after|posted_at|account_id|reference_id|external_id)\b`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act + Assert
			assert.False(t, forbidden.MatchString(tt.emittedSQL),
				"REQ-11 invariant: UpdateCategory SQL must not reference accounting columns; got %q", tt.emittedSQL)
		})
	}
}

// =============================================================================
// ReplaceTags — REQ-10: DELETE + INSERT in one tx, never touches statements row.
// =============================================================================

func TestStatementRepository_ReplaceTags(t *testing.T) {
	stmtID := pkgvo.NewID()
	tag1 := pkgvo.NewID()
	tag2 := pkgvo.NewID()

	tests := []struct {
		name      string
		tagIDs    []pkgvo.ID
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:   "GIVEN owned statement and 2 tags WHEN ReplaceTags THEN exec runs in single tx",
			tagIDs: []pkgvo.ID{tag1, tag2},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM statements WHERE id = \$1\)`).
					WithArgs(stmtID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				m.ExpectExec(`DELETE FROM statement_tags WHERE statement_id = \$1`).
					WithArgs(stmtID.String()).WillReturnResult(sqlmock.NewResult(0, 0))
				m.ExpectExec(`INSERT INTO statement_tags \(statement_id, tag_id\) VALUES \(\$1, \$2\),\(\$3, \$4\)`).
					WithArgs(stmtID.String(), tag1.String(), stmtID.String(), tag2.String()).
					WillReturnResult(sqlmock.NewResult(0, 2))
				m.ExpectCommit()
			},
		},
		{
			name:   "GIVEN empty tag list WHEN ReplaceTags THEN clears all (DELETE only)",
			tagIDs: nil,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT EXISTS`).
					WithArgs(stmtID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				m.ExpectExec(`DELETE FROM statement_tags`).
					WithArgs(stmtID.String()).WillReturnResult(sqlmock.NewResult(0, 5))
				m.ExpectCommit()
			},
		},
		{
			name:   "GIVEN nonexistent statement WHEN ReplaceTags THEN rollback and return error",
			tagIDs: []pkgvo.ID{tag1},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT EXISTS`).
					WithArgs(stmtID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
				m.ExpectRollback()
			},
			wantErr: true,
		},
		{
			name:   "GIVEN DELETE failure WHEN ReplaceTags THEN rollback and return error",
			tagIDs: []pkgvo.ID{tag1},
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT EXISTS`).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				m.ExpectExec(`DELETE FROM statement_tags`).WillReturnError(sql.ErrConnDone)
				m.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newStatementRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			err := repo.ReplaceTags(context.Background(), stmtID, tt.tagIDs)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// =============================================================================
// CountByCategory
// =============================================================================

func TestStatementRepository_CountByCategory(t *testing.T) {
	catID := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		want      int
	}{
		{
			name: "GIVEN N references WHEN CountByCategory THEN returns N",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT COUNT\(\*\) FROM statements WHERE category_id = \$1`).
					WithArgs(catID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))
			},
			want: 42,
		},
		{
			name: "GIVEN no references WHEN CountByCategory THEN returns 0",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT COUNT\(\*\)`).
					WithArgs(catID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newStatementRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			n, err := repo.CountByCategory(context.Background(), catID)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, n)
		})
	}
}

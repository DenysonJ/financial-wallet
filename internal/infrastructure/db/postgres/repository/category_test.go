package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	categorydomain "github.com/DenysonJ/financial-wallet/internal/domain/category"
	categoryvo "github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

var categoryDBColumns = []string{"id", "user_id", "name", "type", "created_at", "updated_at"}

func newCategoryRepoWithMock(t *testing.T) (*CategoryRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, mockErr := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, mockErr)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewCategoryRepository(sqlxDB, sqlxDB)
	return repo, mock, func() { db.Close() }
}

// =============================================================================
// Conversion (no DB)
// =============================================================================

func TestCategoryDB_ToCategory(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	uid := pkgvo.NewID()

	tests := []struct {
		name         string
		input        categoryDB
		wantIsSystem bool
		wantUserID   *pkgvo.ID
		wantType     categoryvo.CategoryType
	}{
		{
			name: "GIVEN user-owned row WHEN toCategory THEN populates UserID and IsSystem=false",
			input: categoryDB{
				ID:        pkgvo.NewID().String(),
				UserID:    sql.NullString{String: uid.String(), Valid: true},
				Name:      "Mercado",
				Type:      "debit",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantIsSystem: false,
			wantUserID:   &uid,
			wantType:     categoryvo.TypeDebit,
		},
		{
			name: "GIVEN system default row (user_id NULL) WHEN toCategory THEN UserID nil and IsSystem=true",
			input: categoryDB{
				ID:        categorydomain.SystemCategoryEstornoCreditID.String(),
				UserID:    sql.NullString{Valid: false},
				Name:      "Estorno",
				Type:      "credit",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantIsSystem: true,
			wantUserID:   nil,
			wantType:     categoryvo.TypeCredit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			c, convErr := tt.input.toCategory()

			// Assert
			require.NoError(t, convErr)
			require.NotNil(t, c)
			assert.Equal(t, tt.wantIsSystem, c.IsSystem())
			assert.Equal(t, tt.wantType, c.Type)
			if tt.wantUserID == nil {
				assert.Nil(t, c.UserID)
			} else {
				require.NotNil(t, c.UserID)
				assert.Equal(t, *tt.wantUserID, *c.UserID)
			}
		})
	}
}

func TestFromDomainCategory(t *testing.T) {
	uid := pkgvo.NewID()

	tests := []struct {
		name          string
		input         *categorydomain.Category
		wantUserValid bool
		wantUserID    string
		wantType      string
	}{
		{
			name:          "GIVEN user-owned domain category WHEN fromDomainCategory THEN UserID is Valid",
			input:         categorydomain.NewCategory(uid, "Mercado", categoryvo.TypeDebit),
			wantUserValid: true,
			wantUserID:    uid.String(),
			wantType:      "debit",
		},
		{
			name:          "GIVEN system default WHEN fromDomainCategory THEN UserID is NULL",
			input:         categorydomain.NewSystemCategory("Estorno", categoryvo.TypeCredit),
			wantUserValid: false,
			wantType:      "credit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			db := fromDomainCategory(tt.input)

			// Assert
			assert.Equal(t, tt.input.ID.String(), db.ID)
			assert.Equal(t, tt.wantUserValid, db.UserID.Valid)
			if tt.wantUserValid {
				assert.Equal(t, tt.wantUserID, db.UserID.String)
			}
			assert.Equal(t, tt.wantType, db.Type)
		})
	}
}

// =============================================================================
// Create
// =============================================================================

func TestCategoryRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "GIVEN valid user-owned input WHEN Create THEN persists",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO categories").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "GIVEN duplicate (user_id, lower(name), type) WHEN Create THEN ErrCategoryDuplicate",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO categories").
					WillReturnError(&pq.Error{Code: pgUniqueViolation, Message: "duplicate key"})
			},
			wantErr: categorydomain.ErrCategoryDuplicate,
		},
		{
			name: "GIVEN unrelated DB error WHEN Create THEN propagates error",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO categories").WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)
			c := categorydomain.NewCategory(pkgvo.NewID(), "Mercado", categoryvo.TypeDebit)

			// Act
			err := repo.Create(context.Background(), c)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// =============================================================================
// FindByID
// =============================================================================

func TestCategoryRepository_FindByID(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	id := pkgvo.NewID()
	uid := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
		wantNil   bool
	}{
		{
			name: "GIVEN existing ID WHEN FindByID THEN returns category",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(id.String(), uid.String(), "Mercado", "debit", now, now)
				m.ExpectQuery("SELECT .+ FROM categories WHERE id").
					WithArgs(id.String()).WillReturnRows(rows)
			},
		},
		{
			name: "GIVEN missing ID WHEN FindByID THEN returns ErrCategoryNotFound",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT .+ FROM categories WHERE id").
					WithArgs(id.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: categorydomain.ErrCategoryNotFound,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			c, findErr := repo.FindByID(context.Background(), id)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, findErr, tt.wantErr)
			} else {
				require.NoError(t, findErr)
			}
			if tt.wantNil {
				assert.Nil(t, c)
			} else {
				assert.NotNil(t, c)
			}
		})
	}
}

// =============================================================================
// FindVisible
// =============================================================================

func TestCategoryRepository_FindVisible(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	id := pkgvo.NewID()
	uid := pkgvo.NewID()

	tests := []struct {
		name         string
		setupMock    func(sqlmock.Sqlmock)
		wantErr      error
		wantIsSystem bool
		wantUserSet  bool
	}{
		{
			name: "GIVEN owned category WHEN FindVisible THEN returns category",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(id.String(), uid.String(), "Mercado", "debit", now, now)
				m.ExpectQuery(`SELECT .+ FROM categories WHERE id = \$1 AND \(user_id = \$2 OR user_id IS NULL\)`).
					WithArgs(id.String(), uid.String()).WillReturnRows(rows)
			},
			wantUserSet: true,
		},
		{
			name: "GIVEN system default WHEN FindVisible THEN returns category with IsSystem=true",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(id.String(), nil, "Estorno", "credit", now, now)
				m.ExpectQuery(`SELECT .+ FROM categories WHERE id`).
					WithArgs(id.String(), uid.String()).WillReturnRows(rows)
			},
			wantIsSystem: true,
		},
		{
			name: "GIVEN cross-user category WHEN FindVisible THEN ErrCategoryNotVisible (not 404)",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT .+ FROM categories WHERE id`).
					WithArgs(id.String(), uid.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: categorydomain.ErrCategoryNotVisible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			c, findErr := repo.FindVisible(context.Background(), id, uid)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, findErr, tt.wantErr)
				assert.Nil(t, c)
				return
			}
			require.NoError(t, findErr)
			require.NotNil(t, c)
			assert.Equal(t, tt.wantIsSystem, c.IsSystem())
			if tt.wantUserSet {
				require.NotNil(t, c.UserID)
				assert.Equal(t, uid, *c.UserID)
			}
		})
	}
}

// =============================================================================
// List
// =============================================================================

func TestCategoryRepository_List(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	uid := pkgvo.NewID()
	credit := categoryvo.TypeCredit

	tests := []struct {
		name         string
		filter       categorydomain.ListFilter
		setupMock    func(sqlmock.Sqlmock)
		wantLen      int
		wantSystemAt []int // indices where IsSystem must be true
	}{
		{
			name:   "GIVEN scope=all WHEN List THEN returns defaults + own",
			filter: categorydomain.ListFilter{UserID: uid},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(pkgvo.NewID().String(), nil, "Salário", "credit", now, now).
					AddRow(pkgvo.NewID().String(), uid.String(), "Mercado", "debit", now, now)
				m.ExpectQuery(`SELECT .+ FROM categories\s+WHERE user_id = \$1 OR user_id IS NULL`).
					WithArgs(uid.String()).WillReturnRows(rows)
			},
			wantLen:      2,
			wantSystemAt: []int{0},
		},
		{
			name:   "GIVEN scope=system WHEN List THEN filters out user categories",
			filter: categorydomain.ListFilter{UserID: uid, Scope: categorydomain.ScopeSystem},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(pkgvo.NewID().String(), nil, "Salário", "credit", now, now)
				m.ExpectQuery(`SELECT .+ FROM categories\s+WHERE user_id IS NULL`).
					WillReturnRows(rows)
			},
			wantLen:      1,
			wantSystemAt: []int{0},
		},
		{
			name:   "GIVEN scope=user WHEN List THEN filters out defaults",
			filter: categorydomain.ListFilter{UserID: uid, Scope: categorydomain.ScopeUser},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(pkgvo.NewID().String(), uid.String(), "Mercado", "debit", now, now)
				m.ExpectQuery(`SELECT .+ FROM categories\s+WHERE user_id = \$1`).
					WithArgs(uid.String()).WillReturnRows(rows)
			},
			wantLen: 1,
		},
		{
			name:   "GIVEN type=credit WHEN List THEN type filter is applied",
			filter: categorydomain.ListFilter{UserID: uid, Type: &credit},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(categoryDBColumns).
					AddRow(pkgvo.NewID().String(), nil, "Salário", "credit", now, now)
				m.ExpectQuery(`SELECT .+ FROM categories\s+WHERE user_id = \$1 OR user_id IS NULL AND type = \$2`).
					WithArgs(uid.String(), "credit").WillReturnRows(rows)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			out, listErr := repo.List(context.Background(), tt.filter)

			// Assert
			require.NoError(t, listErr)
			require.Len(t, out, tt.wantLen)
			for _, idx := range tt.wantSystemAt {
				assert.True(t, out[idx].IsSystem(), "expected IsSystem at index %d", idx)
			}
		})
	}
}

// =============================================================================
// Update
// =============================================================================

func TestCategoryRepository_Update(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock, *categorydomain.Category)
		wantErr   error
	}{
		{
			name: "GIVEN existing owned category WHEN Update THEN succeeds",
			setupMock: func(m sqlmock.Sqlmock, c *categorydomain.Category) {
				m.ExpectExec(`UPDATE categories\s+SET name = \$1, updated_at = \$2\s+WHERE id = \$3`).
					WithArgs(c.Name, c.UpdatedAt, c.ID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "GIVEN nonexistent ID WHEN Update THEN ErrCategoryNotFound",
			setupMock: func(m sqlmock.Sqlmock, _ *categorydomain.Category) {
				m.ExpectExec(`UPDATE categories`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: categorydomain.ErrCategoryNotFound,
		},
		{
			name: "GIVEN rename collision WHEN Update THEN ErrCategoryDuplicate",
			setupMock: func(m sqlmock.Sqlmock, _ *categorydomain.Category) {
				m.ExpectExec(`UPDATE categories`).
					WillReturnError(&pq.Error{Code: pgUniqueViolation})
			},
			wantErr: categorydomain.ErrCategoryDuplicate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			c := categorydomain.NewCategory(pkgvo.NewID(), "Old", categoryvo.TypeDebit)
			require.NoError(t, c.Rename("New"))
			tt.setupMock(mock, c)

			// Act
			err := repo.Update(context.Background(), c)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Delete
// =============================================================================

func TestCategoryRepository_Delete(t *testing.T) {
	id := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "GIVEN existing unused category WHEN Delete THEN succeeds",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM categories WHERE id = \$1`).
					WithArgs(id.String()).WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "GIVEN nonexistent ID WHEN Delete THEN ErrCategoryNotFound",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM categories`).
					WithArgs(id.String()).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: categorydomain.ErrCategoryNotFound,
		},
		{
			name: "GIVEN FK violation (in-use race) WHEN Delete THEN ErrCategoryInUse",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM categories`).
					WithArgs(id.String()).
					WillReturnError(&pq.Error{Code: pgForeignKeyViolation, Message: "FK violation"})
			},
			wantErr: categorydomain.ErrCategoryInUse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			err := repo.Delete(context.Background(), id)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// CountStatementsUsing
// =============================================================================

func TestCategoryRepository_CountStatementsUsing(t *testing.T) {
	id := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		want      int
	}{
		{
			name: "GIVEN N references WHEN CountStatementsUsing THEN returns N",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(7)
				m.ExpectQuery(`SELECT COUNT\(\*\) FROM statements WHERE category_id = \$1`).
					WithArgs(id.String()).WillReturnRows(rows)
			},
			want: 7,
		},
		{
			name: "GIVEN no references WHEN CountStatementsUsing THEN returns 0",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				m.ExpectQuery(`SELECT COUNT\(\*\) FROM statements`).
					WithArgs(id.String()).WillReturnRows(rows)
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newCategoryRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			n, countErr := repo.CountStatementsUsing(context.Background(), id)

			// Assert
			require.NoError(t, countErr)
			assert.Equal(t, tt.want, n)
		})
	}
}

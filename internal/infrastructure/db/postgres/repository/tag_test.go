package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	tagdomain "github.com/DenysonJ/financial-wallet/internal/domain/tag"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

var tagDBColumns = []string{"id", "user_id", "name", "created_at", "updated_at"}

func newTagRepoWithMock(t *testing.T) (*TagRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, mockErr := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, mockErr)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewTagRepository(sqlxDB, sqlxDB)
	return repo, mock, func() { db.Close() }
}

// =============================================================================
// Conversion (no DB)
// =============================================================================

func TestTagDB_ToTag(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	uid := pkgvo.NewID()

	tests := []struct {
		name         string
		input        tagDB
		wantIsSystem bool
		wantUserID   *pkgvo.ID
	}{
		{
			name: "GIVEN user-owned row WHEN toTag THEN populates UserID and IsSystem=false",
			input: tagDB{
				ID:        pkgvo.NewID().String(),
				UserID:    sql.NullString{String: uid.String(), Valid: true},
				Name:      "viagem-2026",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantIsSystem: false,
			wantUserID:   &uid,
		},
		{
			name: "GIVEN system default row (user_id NULL) WHEN toTag THEN UserID nil and IsSystem=true",
			input: tagDB{
				ID:        pkgvo.NewID().String(),
				UserID:    sql.NullString{Valid: false},
				Name:      "recorrente",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantIsSystem: true,
			wantUserID:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			tag, convErr := tt.input.toTag()

			// Assert
			require.NoError(t, convErr)
			require.NotNil(t, tag)
			assert.Equal(t, tt.wantIsSystem, tag.IsSystem())
			if tt.wantUserID == nil {
				assert.Nil(t, tag.UserID)
			} else {
				require.NotNil(t, tag.UserID)
				assert.Equal(t, *tt.wantUserID, *tag.UserID)
			}
		})
	}
}

func TestFromDomainTag(t *testing.T) {
	uid := pkgvo.NewID()

	tests := []struct {
		name          string
		input         *tagdomain.Tag
		wantUserValid bool
		wantUserID    string
	}{
		{
			name:          "GIVEN user-owned domain tag WHEN fromDomainTag THEN UserID is Valid",
			input:         tagdomain.NewTag(uid, "viagem"),
			wantUserValid: true,
			wantUserID:    uid.String(),
		},
		{
			name:          "GIVEN system default WHEN fromDomainTag THEN UserID is NULL",
			input:         tagdomain.NewSystemTag("recorrente"),
			wantUserValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			db := fromDomainTag(tt.input)

			// Assert
			assert.Equal(t, tt.wantUserValid, db.UserID.Valid)
			if tt.wantUserValid {
				assert.Equal(t, tt.wantUserID, db.UserID.String)
			}
		})
	}
}

// =============================================================================
// Create
// =============================================================================

func TestTagRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "GIVEN valid input WHEN Create THEN persists",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO tags").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "GIVEN duplicate (user_id, lower(name)) WHEN Create THEN ErrTagDuplicate",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO tags").
					WillReturnError(&pq.Error{Code: pgUniqueViolation})
			},
			wantErr: tagdomain.ErrTagDuplicate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)
			tag := tagdomain.NewTag(pkgvo.NewID(), "viagem")

			// Act
			err := repo.Create(context.Background(), tag)

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
// FindByID
// =============================================================================

func TestTagRepository_FindByID(t *testing.T) {
	id := pkgvo.NewID()
	uid := pkgvo.NewID()
	now := time.Now().Truncate(time.Microsecond)

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
		wantNil   bool
	}{
		{
			name: "GIVEN existing ID WHEN FindByID THEN returns tag",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(id.String(), uid.String(), "viagem", now, now)
				m.ExpectQuery("SELECT .+ FROM tags WHERE id").
					WithArgs(id.String()).WillReturnRows(rows)
			},
		},
		{
			name: "GIVEN missing ID WHEN FindByID THEN returns ErrTagNotFound",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT .+ FROM tags WHERE id").
					WithArgs(id.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: tagdomain.ErrTagNotFound,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			tag, findErr := repo.FindByID(context.Background(), id)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, findErr, tt.wantErr)
			} else {
				require.NoError(t, findErr)
			}
			if tt.wantNil {
				assert.Nil(t, tag)
			} else {
				assert.NotNil(t, tag)
			}
		})
	}
}

// =============================================================================
// FindVisible
// =============================================================================

func TestTagRepository_FindVisible(t *testing.T) {
	id := pkgvo.NewID()
	uid := pkgvo.NewID()
	now := time.Now().Truncate(time.Microsecond)

	tests := []struct {
		name         string
		setupMock    func(sqlmock.Sqlmock)
		wantErr      error
		wantIsSystem bool
		wantUserSet  bool
	}{
		{
			name: "GIVEN owned tag WHEN FindVisible THEN returns tag",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(id.String(), uid.String(), "viagem", now, now)
				m.ExpectQuery(`SELECT .+ FROM tags WHERE id = \$1 AND \(user_id = \$2 OR user_id IS NULL\)`).
					WithArgs(id.String(), uid.String()).WillReturnRows(rows)
			},
			wantUserSet: true,
		},
		{
			name: "GIVEN system default WHEN FindVisible THEN returns tag with IsSystem=true",
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(id.String(), nil, "recorrente", now, now)
				m.ExpectQuery(`SELECT .+ FROM tags WHERE id`).
					WithArgs(id.String(), uid.String()).WillReturnRows(rows)
			},
			wantIsSystem: true,
		},
		{
			name: "GIVEN cross-user tag WHEN FindVisible THEN ErrTagNotVisible (not 404)",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT .+ FROM tags WHERE id`).
					WithArgs(id.String(), uid.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: tagdomain.ErrTagNotVisible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			tag, findErr := repo.FindVisible(context.Background(), id, uid)

			// Assert
			if tt.wantErr != nil {
				assert.ErrorIs(t, findErr, tt.wantErr)
				assert.Nil(t, tag)
				return
			}
			require.NoError(t, findErr)
			require.NotNil(t, tag)
			assert.Equal(t, tt.wantIsSystem, tag.IsSystem())
			if tt.wantUserSet {
				require.NotNil(t, tag.UserID)
			}
		})
	}
}

// =============================================================================
// FindManyVisible
// =============================================================================

func TestTagRepository_FindManyVisible(t *testing.T) {
	uid := pkgvo.NewID()
	id1 := pkgvo.NewID()
	id2 := pkgvo.NewID()
	now := time.Now().Truncate(time.Microsecond)

	tests := []struct {
		name      string
		ids       []pkgvo.ID
		setupMock func(sqlmock.Sqlmock)
		wantLen   int
	}{
		{
			name:      "GIVEN empty input WHEN FindManyVisible THEN returns empty slice without DB call",
			ids:       nil,
			setupMock: func(_ sqlmock.Sqlmock) {},
			wantLen:   0,
		},
		{
			name: "GIVEN 3 IDs (1 cross-user) WHEN FindManyVisible THEN returns the 2 visible ones",
			ids:  []pkgvo.ID{id1, id2, pkgvo.NewID()},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(id1.String(), uid.String(), "viagem", now, now).
					AddRow(id2.String(), nil, "recorrente", now, now)
				m.ExpectQuery(`SELECT .+ FROM tags\s+WHERE id = ANY\(\$1\) AND \(user_id = \$2 OR user_id IS NULL\)`).
					WillReturnRows(rows)
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
			defer cleanup()
			tt.setupMock(mock)

			// Act
			out, findErr := repo.FindManyVisible(context.Background(), tt.ids, uid)

			// Assert
			require.NoError(t, findErr)
			assert.Len(t, out, tt.wantLen)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// =============================================================================
// List
// =============================================================================

func TestTagRepository_List(t *testing.T) {
	uid := pkgvo.NewID()
	now := time.Now().Truncate(time.Microsecond)

	tests := []struct {
		name         string
		filter       tagdomain.ListFilter
		setupMock    func(sqlmock.Sqlmock)
		wantLen      int
		wantSystemAt []int
	}{
		{
			name:   "GIVEN scope=all WHEN List THEN returns defaults + own",
			filter: tagdomain.ListFilter{UserID: uid},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(pkgvo.NewID().String(), nil, "recorrente", now, now).
					AddRow(pkgvo.NewID().String(), uid.String(), "viagem", now, now)
				m.ExpectQuery(`SELECT .+ FROM tags\s+WHERE user_id = \$1 OR user_id IS NULL`).
					WithArgs(uid.String()).WillReturnRows(rows)
			},
			wantLen:      2,
			wantSystemAt: []int{0},
		},
		{
			name:   "GIVEN scope=system WHEN List THEN filters out user tags",
			filter: tagdomain.ListFilter{UserID: uid, Scope: tagdomain.ScopeSystem},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(pkgvo.NewID().String(), nil, "recorrente", now, now)
				m.ExpectQuery(`SELECT .+ FROM tags\s+WHERE user_id IS NULL`).WillReturnRows(rows)
			},
			wantLen:      1,
			wantSystemAt: []int{0},
		},
		{
			name:   "GIVEN scope=user WHEN List THEN filters out defaults",
			filter: tagdomain.ListFilter{UserID: uid, Scope: tagdomain.ScopeUser},
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(tagDBColumns).
					AddRow(pkgvo.NewID().String(), uid.String(), "viagem", now, now)
				m.ExpectQuery(`SELECT .+ FROM tags\s+WHERE user_id = \$1`).
					WithArgs(uid.String()).WillReturnRows(rows)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
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

func TestTagRepository_Update(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock, *tagdomain.Tag)
		wantErr   error
	}{
		{
			name: "GIVEN existing owned tag WHEN Update THEN succeeds",
			setupMock: func(m sqlmock.Sqlmock, tag *tagdomain.Tag) {
				m.ExpectExec(`UPDATE tags\s+SET name = \$1, updated_at = \$2\s+WHERE id = \$3`).
					WithArgs(tag.Name, tag.UpdatedAt, tag.ID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "GIVEN nonexistent ID WHEN Update THEN ErrTagNotFound",
			setupMock: func(m sqlmock.Sqlmock, _ *tagdomain.Tag) {
				m.ExpectExec(`UPDATE tags`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: tagdomain.ErrTagNotFound,
		},
		{
			name: "GIVEN rename collision WHEN Update THEN ErrTagDuplicate",
			setupMock: func(m sqlmock.Sqlmock, _ *tagdomain.Tag) {
				m.ExpectExec(`UPDATE tags`).WillReturnError(&pq.Error{Code: pgUniqueViolation})
			},
			wantErr: tagdomain.ErrTagDuplicate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
			defer cleanup()
			tag := tagdomain.NewTag(pkgvo.NewID(), "old")
			require.NoError(t, tag.Rename("new"))
			tt.setupMock(mock, tag)

			// Act
			err := repo.Update(context.Background(), tag)

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

func TestTagRepository_Delete(t *testing.T) {
	id := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "GIVEN existing tag WHEN Delete THEN succeeds (CASCADE handles statement_tags)",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM tags WHERE id = \$1`).
					WithArgs(id.String()).WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "GIVEN nonexistent ID WHEN Delete THEN ErrTagNotFound",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`DELETE FROM tags`).
					WithArgs(id.String()).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: tagdomain.ErrTagNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo, mock, cleanup := newTagRepoWithMock(t)
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

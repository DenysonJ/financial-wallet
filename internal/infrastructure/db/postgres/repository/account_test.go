package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/internal/domain/user/vo"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

func buildTestAccount() *accountdomain.Account {
	now := time.Now().Truncate(time.Microsecond)
	return &accountdomain.Account{
		ID:          uservo.NewID(),
		UserID:      uservo.NewID(),
		Name:        "Nubank",
		Type:        accountvo.TypeBankAccount,
		Description: "Conta corrente",
		Active:      true,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now,
	}
}

var accountDBColumns = []string{"id", "user_id", "name", "type", "description", "active", "created_at", "updated_at"}

// =============================================================================
// Unit Tests for internal conversions
// =============================================================================

func TestAccountDB_ToAccount(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)

	tests := []struct {
		name      string
		input     accountDB
		wantErr   bool
		errSubstr string
	}{
		{
			name: "sucesso com todos os campos",
			input: accountDB{
				ID:          "018e4a2c-6b4d-7000-9410-abcdef123456",
				UserID:      "018e4a2c-6b4d-7000-9410-abcdef654321",
				Name:        "Nubank",
				Type:        "bank_account",
				Description: "Conta corrente",
				Active:      true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: false,
		},
		{
			name: "ID inválido",
			input: accountDB{
				ID:     "invalid-id",
				UserID: "018e4a2c-6b4d-7000-9410-abcdef654321",
				Name:   "Test",
				Type:   "cash",
			},
			wantErr:   true,
			errSubstr: "parsing account ID",
		},
		{
			name: "UserID inválido",
			input: accountDB{
				ID:     "018e4a2c-6b4d-7000-9410-abcdef123456",
				UserID: "invalid-user-id",
				Name:   "Test",
				Type:   "cash",
			},
			wantErr:   true,
			errSubstr: "parsing user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, convertErr := tt.input.toAccount()

			if tt.wantErr {
				assert.Error(t, convertErr)
				assert.Nil(t, result)
				assert.Contains(t, convertErr.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, convertErr)
				require.NotNil(t, result)
				assert.Equal(t, tt.input.ID, result.ID.String())
				assert.Equal(t, tt.input.UserID, result.UserID.String())
				assert.Equal(t, tt.input.Name, result.Name)
				assert.Equal(t, tt.input.Description, result.Description)
				assert.Equal(t, tt.input.Active, result.Active)
			}
		})
	}
}

func TestFromDomainAccount_RoundTrip(t *testing.T) {
	original := buildTestAccount()

	dbModel := fromDomainAccount(original)
	restored, convertErr := dbModel.toAccount()

	assert.NoError(t, convertErr)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.UserID, restored.UserID)
	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.Description, restored.Description)
	assert.Equal(t, original.Active, restored.Active)
	assert.Equal(t, original.CreatedAt, restored.CreatedAt)
	assert.Equal(t, original.UpdatedAt, restored.UpdatedAt)
}

// =============================================================================
// Unit Tests for AccountRepository with sqlmock
// =============================================================================

// --- Create ------------------------------------------------------------------

func TestAccountRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{
			name:    "sucesso",
			dbErr:   nil,
			wantErr: nil,
		},
		{
			name:    "erro de banco",
			dbErr:   sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewAccountRepository(sqlxDB, sqlxDB)

			exec := mock.ExpectExec("INSERT INTO accounts").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg())

			if tt.dbErr != nil {
				exec.WillReturnError(tt.dbErr)
			} else {
				exec.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			createErr := repo.Create(context.Background(), buildTestAccount())

			if tt.wantErr != nil {
				assert.ErrorIs(t, createErr, tt.wantErr)
			} else {
				assert.NoError(t, createErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- FindByID ----------------------------------------------------------------

func TestAccountRepository_FindByID(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	testID := uservo.NewID()
	testUserID := uservo.NewID()

	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   error
		wantNil   bool
		wantName  string
	}{
		{
			name: "sucesso",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(accountDBColumns).
					AddRow(testID.String(), testUserID.String(), "Nubank", "bank_account", "Conta corrente", true, now, now)
				mock.ExpectQuery("SELECT .+ FROM accounts WHERE id").
					WithArgs(testID.String()).WillReturnRows(rows)
			},
			wantNil:  false,
			wantName: "Nubank",
		},
		{
			name: "não encontrado",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM accounts WHERE id").
					WithArgs(testID.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: accountdomain.ErrAccountNotFound,
			wantNil: true,
		},
		{
			name: "erro de banco",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM accounts WHERE id").
					WithArgs(testID.String()).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewAccountRepository(sqlxDB, sqlxDB)
			tt.setupMock(mock)

			result, findErr := repo.FindByID(context.Background(), testID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, findErr, tt.wantErr)
			} else {
				assert.NoError(t, findErr)
			}
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.wantName, result.Name)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- List --------------------------------------------------------------------

func TestAccountRepository_List(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	testID := uservo.NewID()
	testUserID := uservo.NewID()

	tests := []struct {
		name      string
		filter    accountdomain.ListFilter
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
		errSubstr string
		wantTotal int
		wantCount int
	}{
		{
			name:   "sucesso com resultados",
			filter: accountdomain.ListFilter{Page: 1, Limit: 20, UserID: testUserID},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM accounts WHERE user_id").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery("SELECT .+ FROM accounts").
					WillReturnRows(sqlmock.NewRows(accountDBColumns).
						AddRow(testID.String(), testUserID.String(), "Nubank", "bank_account", "", true, now, now))
				mock.ExpectCommit()
			},
			wantTotal: 1,
			wantCount: 1,
		},
		{
			name:   "resultado vazio",
			filter: accountdomain.ListFilter{Page: 1, Limit: 20, UserID: testUserID},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM accounts WHERE user_id").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery("SELECT .+ FROM accounts").
					WillReturnRows(sqlmock.NewRows(accountDBColumns))
				mock.ExpectCommit()
			},
			wantTotal: 0,
			wantCount: 0,
		},
		{
			name:      "user_id vazio é rejeitado",
			filter:    accountdomain.ListFilter{Page: 1, Limit: 20, UserID: ""},
			setupMock: func(_ sqlmock.Sqlmock) {},
			wantErr:   true,
			errSubstr: "user_id is required",
		},
		{
			name:   "com filtro de tipo",
			filter: accountdomain.ListFilter{Page: 1, Limit: 20, UserID: testUserID, Type: "bank_account"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM accounts WHERE user_id.+AND type").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery("SELECT .+ FROM accounts.+WHERE user_id.+AND type").
					WillReturnRows(sqlmock.NewRows(accountDBColumns).
						AddRow(testID.String(), testUserID.String(), "Nubank", "bank_account", "", true, now, now))
				mock.ExpectCommit()
			},
			wantTotal: 1,
			wantCount: 1,
		},
		{
			name:   "erro ao iniciar transação",
			filter: accountdomain.ListFilter{Page: 1, Limit: 20, UserID: testUserID},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(sql.ErrConnDone)
			},
			wantErr:   true,
			errSubstr: "beginning read transaction",
		},
		{
			name:   "erro na query de contagem",
			filter: accountdomain.ListFilter{Page: 1, Limit: 20, UserID: testUserID},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM accounts").
					WillReturnError(sql.ErrConnDone)
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewAccountRepository(sqlxDB, sqlxDB)
			tt.setupMock(mock)

			result, listErr := repo.List(context.Background(), tt.filter)

			if tt.wantErr {
				assert.Error(t, listErr)
				assert.Nil(t, result)
				if tt.errSubstr != "" {
					assert.Contains(t, listErr.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, listErr)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantTotal, result.Total)
				assert.Len(t, result.Accounts, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- Update ------------------------------------------------------------------

func TestAccountRepository_Update(t *testing.T) {
	tests := []struct {
		name      string
		dbResult  sql.Result
		dbErr     error
		wantErr   error
		errSubstr string
	}{
		{
			name:     "sucesso",
			dbResult: sqlmock.NewResult(0, 1),
		},
		{
			name:     "não encontrado - zero rows affected",
			dbResult: sqlmock.NewResult(0, 0),
			wantErr:  accountdomain.ErrAccountNotFound,
		},
		{
			name:    "erro de banco",
			dbErr:   sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewAccountRepository(sqlxDB, sqlxDB)

			exec := mock.ExpectExec("UPDATE accounts SET").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg())

			if tt.dbErr != nil {
				exec.WillReturnError(tt.dbErr)
			} else {
				exec.WillReturnResult(tt.dbResult)
			}

			updateErr := repo.Update(context.Background(), buildTestAccount())

			if tt.wantErr != nil {
				assert.ErrorIs(t, updateErr, tt.wantErr)
			} else {
				assert.NoError(t, updateErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- Delete ------------------------------------------------------------------

func TestAccountRepository_Delete(t *testing.T) {
	testID := uservo.NewID()

	tests := []struct {
		name     string
		dbResult sql.Result
		dbErr    error
		wantErr  error
	}{
		{
			name:     "sucesso",
			dbResult: sqlmock.NewResult(0, 1),
		},
		{
			name:     "não encontrado - zero rows affected",
			dbResult: sqlmock.NewResult(0, 0),
			wantErr:  accountdomain.ErrAccountNotFound,
		},
		{
			name:    "erro de banco",
			dbErr:   sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewAccountRepository(sqlxDB, sqlxDB)

			exec := mock.ExpectExec("UPDATE accounts SET").
				WithArgs(sqlmock.AnyArg(), testID.String())

			if tt.dbErr != nil {
				exec.WillReturnError(tt.dbErr)
			} else {
				exec.WillReturnResult(tt.dbResult)
			}

			deleteErr := repo.Delete(context.Background(), testID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, deleteErr, tt.wantErr)
			} else {
				assert.NoError(t, deleteErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

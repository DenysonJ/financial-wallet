package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	stmtvo "github.com/DenysonJ/financial-wallet/internal/domain/statement/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

func buildTestStatement() *stmtdomain.Statement {
	now := time.Now().Truncate(time.Microsecond)
	amount, _ := stmtvo.NewAmount(5000)
	return &stmtdomain.Statement{
		ID:           pkgvo.NewID(),
		AccountID:    pkgvo.NewID(),
		Type:         stmtvo.TypeCredit,
		Amount:       amount,
		Description:  "Test deposit",
		BalanceAfter: 0,
		CreatedAt:    now,
	}
}

func buildTestReversalStatement(referenceID pkgvo.ID) *stmtdomain.Statement {
	now := time.Now().Truncate(time.Microsecond)
	amount, _ := stmtvo.NewAmount(5000)
	return &stmtdomain.Statement{
		ID:           pkgvo.NewID(),
		AccountID:    pkgvo.NewID(),
		Type:         stmtvo.TypeDebit,
		Amount:       amount,
		Description:  "Reversal",
		ReferenceID:  &referenceID,
		BalanceAfter: 0,
		CreatedAt:    now,
	}
}

var statementDBColumns = []string{"id", "account_id", "type", "amount", "description", "reference_id", "balance_after", "created_at"}

// =============================================================================
// Unit Tests for internal conversions
// =============================================================================

func TestStatementDB_ToStatement(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	refID := "018e4a2c-6b4d-7000-9410-abcdef999999"

	tests := []struct {
		name      string
		input     statementDB
		wantErr   bool
		errSubstr string
	}{
		{
			name: "given valid data without reference_id when converting then succeeds",
			input: statementDB{
				ID: "018e4a2c-6b4d-7000-9410-abcdef123456", AccountID: "018e4a2c-6b4d-7000-9410-abcdef654321",
				Type: "credit", Amount: 5000, Description: "Salary",
				BalanceAfter: 15000, CreatedAt: now,
			},
		},
		{
			name: "given valid data with reference_id when converting then succeeds",
			input: statementDB{
				ID: "018e4a2c-6b4d-7000-9410-abcdef123456", AccountID: "018e4a2c-6b4d-7000-9410-abcdef654321",
				Type: "debit", Amount: 5000, Description: "Reversal",
				ReferenceID: &refID, BalanceAfter: 10000, CreatedAt: now,
			},
		},
		{
			name: "given invalid ID when converting then returns error",
			input: statementDB{
				ID: "invalid-id", AccountID: "018e4a2c-6b4d-7000-9410-abcdef654321",
				Type: "credit", Amount: 1000,
			},
			wantErr:   true,
			errSubstr: "parsing statement ID",
		},
		{
			name: "given invalid AccountID when converting then returns error",
			input: statementDB{
				ID: "018e4a2c-6b4d-7000-9410-abcdef123456", AccountID: "invalid",
				Type: "credit", Amount: 1000,
			},
			wantErr:   true,
			errSubstr: "parsing account ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, convertErr := tt.input.toStatement()

			if tt.wantErr {
				assert.Error(t, convertErr)
				assert.Nil(t, result)
				assert.Contains(t, convertErr.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, convertErr)
				require.NotNil(t, result)
				assert.Equal(t, tt.input.ID, result.ID.String())
				assert.Equal(t, tt.input.AccountID, result.AccountID.String())
				assert.Equal(t, tt.input.Amount, result.Amount.Int64())
			}
		})
	}
}

func TestFromDomainStatement_RoundTrip(t *testing.T) {
	original := buildTestStatement()
	original.SetBalanceAfter(15000)

	dbModel := fromDomainStatement(original)
	restored, convertErr := dbModel.toStatement()

	assert.NoError(t, convertErr)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.AccountID, restored.AccountID)
	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.Amount, restored.Amount)
	assert.Equal(t, original.Description, restored.Description)
	assert.Equal(t, original.BalanceAfter, restored.BalanceAfter)
	assert.Equal(t, original.CreatedAt, restored.CreatedAt)
	assert.Nil(t, restored.ReferenceID)
}

func TestFromDomainStatement_RoundTrip_WithReferenceID(t *testing.T) {
	refID := pkgvo.NewID()
	original := buildTestReversalStatement(refID)
	original.SetBalanceAfter(10000)

	dbModel := fromDomainStatement(original)
	restored, convertErr := dbModel.toStatement()

	assert.NoError(t, convertErr)
	require.NotNil(t, restored.ReferenceID)
	assert.Equal(t, refID, *restored.ReferenceID)
}

// =============================================================================
// Unit Tests for StatementRepository with sqlmock
// =============================================================================

// --- Create ------------------------------------------------------------------

func TestStatementRepository_Create(t *testing.T) {
	accountID := pkgvo.NewID()

	tests := []struct {
		name        string
		stmtType    stmtvo.StatementType
		balance     int64
		amount      int64
		lockErr     error
		insertErr   error
		updateErr   error
		commitErr   error
		wantErr     bool
		errSubstr   string
		wantBalance int64
	}{
		{
			name:     "given credit when creating then increases balance",
			stmtType: stmtvo.TypeCredit, balance: 10000, amount: 5000,
			wantBalance: 15000,
		},
		{
			name:     "given debit when creating then decreases balance",
			stmtType: stmtvo.TypeDebit, balance: 10000, amount: 3000,
			wantBalance: 7000,
		},
		{
			name:     "given debit exceeding balance when creating then allows negative",
			stmtType: stmtvo.TypeDebit, balance: 1000, amount: 5000,
			wantBalance: -4000,
		},
		{
			name:     "given lock failure when creating then returns error",
			stmtType: stmtvo.TypeCredit, balance: 0, amount: 1000,
			lockErr: sql.ErrConnDone,
			wantErr: true, errSubstr: "locking account",
		},
		{
			name:     "given nonexistent account when locking then returns not found",
			stmtType: stmtvo.TypeCredit, balance: 0, amount: 1000,
			lockErr: sql.ErrNoRows,
			wantErr: true, errSubstr: "account not found",
		},
		{
			name:     "given insert failure when creating then returns error",
			stmtType: stmtvo.TypeCredit, balance: 10000, amount: 1000,
			insertErr: sql.ErrConnDone,
			wantErr:   true, errSubstr: "inserting statement",
		},
		{
			name:     "given update failure when creating then returns error",
			stmtType: stmtvo.TypeCredit, balance: 10000, amount: 1000,
			updateErr: sql.ErrConnDone,
			wantErr:   true, errSubstr: "updating account balance",
		},
		{
			name:     "given commit failure when creating then returns error",
			stmtType: stmtvo.TypeCredit, balance: 10000, amount: 1000,
			commitErr: sql.ErrConnDone,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewStatementRepository(sqlxDB, sqlxDB)

			amount, _ := stmtvo.NewAmount(tt.amount)
			stmt := &stmtdomain.Statement{
				ID: pkgvo.NewID(), AccountID: accountID, Type: tt.stmtType,
				Amount: amount, Description: "Test", CreatedAt: time.Now(),
			}

			mock.ExpectBegin()

			// Lock query
			lockQuery := mock.ExpectQuery("SELECT balance FROM accounts WHERE id").
				WithArgs(accountID.String())
			if tt.lockErr != nil {
				lockQuery.WillReturnError(tt.lockErr)
				mock.ExpectRollback()
			} else {
				lockQuery.WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(tt.balance))

				// Insert
				insertExec := mock.ExpectExec("INSERT INTO statements").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg())
				if tt.insertErr != nil {
					insertExec.WillReturnError(tt.insertErr)
					mock.ExpectRollback()
				} else {
					insertExec.WillReturnResult(sqlmock.NewResult(0, 1))

					// Update balance
					updateExec := mock.ExpectExec("UPDATE accounts SET balance").
						WithArgs(sqlmock.AnyArg(), accountID.String())
					if tt.updateErr != nil {
						updateExec.WillReturnError(tt.updateErr)
						mock.ExpectRollback()
					} else {
						updateExec.WillReturnResult(sqlmock.NewResult(0, 1))
						if tt.commitErr != nil {
							mock.ExpectCommit().WillReturnError(tt.commitErr)
						} else {
							mock.ExpectCommit()
						}
					}
				}
			}

			createErr := repo.Create(context.Background(), stmt, accountID)

			if tt.wantErr {
				assert.Error(t, createErr)
				assert.Contains(t, createErr.Error(), tt.errSubstr)
			} else {
				assert.NoError(t, createErr)
				assert.Equal(t, tt.wantBalance, stmt.BalanceAfter)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- FindByID ----------------------------------------------------------------

func TestStatementRepository_FindByID(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	testID := pkgvo.NewID()
	testAccountID := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   error
		wantNil   bool
	}{
		{
			name: "given existing statement when finding by ID then returns it",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(statementDBColumns).
					AddRow(testID.String(), testAccountID.String(), "credit", int64(5000), "Salary", nil, int64(15000), now)
				mock.ExpectQuery("SELECT .+ FROM statements WHERE id").
					WithArgs(testID.String()).WillReturnRows(rows)
			},
		},
		{
			name: "given nonexistent ID when finding then returns not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM statements WHERE id").
					WithArgs(testID.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: stmtdomain.ErrStatementNotFound,
			wantNil: true,
		},
		{
			name: "given db failure when querying then returns error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM statements WHERE id").
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
			repo := NewStatementRepository(sqlxDB, sqlxDB)
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
				assert.Equal(t, testID, result.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- List --------------------------------------------------------------------

func TestStatementRepository_List(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	testID := pkgvo.NewID()
	testAccountID := pkgvo.NewID()

	tests := []struct {
		name      string
		filter    stmtdomain.ListFilter
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
		errSubstr string
		wantTotal int
		wantCount int
	}{
		{
			name:   "given statements when listing then returns paginated results",
			filter: stmtdomain.ListFilter{AccountID: testAccountID, Page: 1, Limit: 20},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM statements WHERE account_id").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery("SELECT .+ FROM statements").
					WillReturnRows(sqlmock.NewRows(statementDBColumns).
						AddRow(testID.String(), testAccountID.String(), "credit", int64(5000), "Salary", nil, int64(15000), now))
				mock.ExpectCommit()
			},
			wantTotal: 1,
			wantCount: 1,
		},
		{
			name:   "given no statements when listing then returns empty",
			filter: stmtdomain.ListFilter{AccountID: testAccountID, Page: 1, Limit: 20},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM statements WHERE account_id").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery("SELECT .+ FROM statements").
					WillReturnRows(sqlmock.NewRows(statementDBColumns))
				mock.ExpectCommit()
			},
			wantTotal: 0,
			wantCount: 0,
		},
		{
			name:   "given tx failure when listing then returns error",
			filter: stmtdomain.ListFilter{AccountID: testAccountID, Page: 1, Limit: 20},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(sql.ErrConnDone)
			},
			wantErr:   true,
			errSubstr: "beginning read transaction",
		},
		{
			name:   "given count query failure when listing then returns error",
			filter: stmtdomain.ListFilter{AccountID: testAccountID, Page: 1, Limit: 20},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM statements").
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
			repo := NewStatementRepository(sqlxDB, sqlxDB)
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
				assert.Len(t, result.Statements, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- HasReversal -------------------------------------------------------------

func TestStatementRepository_HasReversal(t *testing.T) {
	testID := pkgvo.NewID()

	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		want      bool
		wantErr   bool
	}{
		{
			name: "given unreversed statement when checking then returns false",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(testID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			want: false,
		},
		{
			name: "given reversed statement when checking then returns true",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(testID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			want: true,
		},
		{
			name: "given db failure when querying then returns error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(testID.String()).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			repo := NewStatementRepository(sqlxDB, sqlxDB)
			tt.setupMock(mock)

			result, hasErr := repo.HasReversal(context.Background(), testID)

			if tt.wantErr {
				assert.Error(t, hasErr)
			} else {
				assert.NoError(t, hasErr)
				assert.Equal(t, tt.want, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

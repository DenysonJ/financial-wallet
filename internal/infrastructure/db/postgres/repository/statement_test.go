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
		PostedAt:     now,
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
		PostedAt:     now,
		CreatedAt:    now,
	}
}

var statementDBColumns = []string{
	"id", "account_id", "type", "amount", "description", "reference_id", "external_id",
	"balance_after", "posted_at", "created_at",
	"category_id", "category_name", "category_type", "tags_json",
}

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

				// Insert (11 columns including category_id)
				insertExec := mock.ExpectExec("INSERT INTO statements").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg())
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

			balanceAfter, createErr := repo.Create(context.Background(), stmt, accountID)

			if tt.wantErr {
				assert.Error(t, createErr)
				assert.Contains(t, createErr.Error(), tt.errSubstr)
				assert.Equal(t, int64(0), balanceAfter)
			} else {
				assert.NoError(t, createErr)
				assert.Equal(t, tt.wantBalance, balanceAfter)
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
					AddRow(testID.String(), testAccountID.String(), "credit", int64(5000), "Salary", nil, nil, int64(15000), now, now, nil, nil, nil, []byte("[]"))
				mock.ExpectQuery(`SELECT .+ FROM statements s\s+LEFT JOIN categories c ON c\.id = s\.category_id\s+WHERE s\.id`).
					WithArgs(testID.String()).WillReturnRows(rows)
			},
		},
		{
			name: "given nonexistent ID when finding then returns not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM statements s\s+LEFT JOIN categories c ON c\.id = s\.category_id\s+WHERE s\.id`).
					WithArgs(testID.String()).WillReturnError(sql.ErrNoRows)
			},
			wantErr: stmtdomain.ErrStatementNotFound,
			wantNil: true,
		},
		{
			name: "given db failure when querying then returns error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM statements s\s+LEFT JOIN categories c ON c\.id = s\.category_id\s+WHERE s\.id`).
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
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM statements s WHERE s\.account_id`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery("SELECT .+ FROM statements").
					WillReturnRows(sqlmock.NewRows(statementDBColumns).
						AddRow(testID.String(), testAccountID.String(), "credit", int64(5000), "Salary", nil, nil, int64(15000), now, now, nil, nil, nil, []byte("[]")))
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
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM statements s WHERE s\.account_id`).
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

// --- CreateBatch -------------------------------------------------------------

func TestStatementRepository_CreateBatch(t *testing.T) {
	accountID := pkgvo.NewID()
	now := time.Now().Truncate(time.Microsecond)

	creditAmount, _ := stmtvo.NewAmount(5000)
	debitAmount, _ := stmtvo.NewAmount(2000)

	creditStmt := &stmtdomain.Statement{
		ID: pkgvo.NewID(), AccountID: accountID, Type: stmtvo.TypeCredit,
		Amount: creditAmount, Description: "Salary", PostedAt: now, CreatedAt: now,
	}
	debitStmt := &stmtdomain.Statement{
		ID: pkgvo.NewID(), AccountID: accountID, Type: stmtvo.TypeDebit,
		Amount: debitAmount, Description: "Uber", PostedAt: now, CreatedAt: now,
	}

	tests := []struct {
		name        string
		stmts       []*stmtdomain.Statement
		setupMock   func(mock sqlmock.Sqlmock)
		wantBalance int64
		wantErr     bool
		errSubstr   string
	}{
		{
			name:  "given two statements when batch creating then updates balance sequentially",
			stmts: []*stmtdomain.Statement{creditStmt, debitStmt},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT balance FROM accounts WHERE id").
					WithArgs(accountID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(int64(10000)))
				mock.ExpectExec("INSERT INTO statements").
					WillReturnResult(sqlmock.NewResult(0, 2))
				mock.ExpectExec("UPDATE accounts SET balance").
					WithArgs(int64(13000), accountID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantBalance: 13000, // 10000 + 5000 - 2000
		},
		{
			name:  "given account not found when batch creating then returns error",
			stmts: []*stmtdomain.Statement{creditStmt},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT balance FROM accounts WHERE id").
					WithArgs(accountID.String()).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr:   true,
			errSubstr: "account not found",
		},
		{
			name:  "given insert failure when batch creating then rolls back",
			stmts: []*stmtdomain.Statement{creditStmt},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT balance FROM accounts WHERE id").
					WithArgs(accountID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(int64(10000)))
				mock.ExpectExec("INSERT INTO statements").
					WillReturnError(sql.ErrConnDone)
				mock.ExpectRollback()
			},
			wantErr:   true,
			errSubstr: "inserting statement batch",
		},
		{
			name:  "given empty batch when creating then returns zero without transaction",
			stmts: []*stmtdomain.Statement{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No DB calls expected for empty batch
			},
			wantBalance: 0,
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

			balance, batchErr := repo.CreateBatch(context.Background(), tt.stmts, accountID)

			if tt.wantErr {
				assert.Error(t, batchErr)
				if tt.errSubstr != "" {
					assert.Contains(t, batchErr.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, batchErr)
				assert.Equal(t, tt.wantBalance, balance)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- FindExternalIDs ---------------------------------------------------------

func TestStatementRepository_FindExternalIDs(t *testing.T) {
	accountID := pkgvo.NewID()

	tests := []struct {
		name        string
		externalIDs []string
		setupMock   func(mock sqlmock.Sqlmock)
		want        map[string]bool
		wantErr     bool
	}{
		{
			name:        "given matching IDs when finding then returns found set",
			externalIDs: []string{"FIT001", "FIT002", "FIT003"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT external_id FROM statements WHERE account_id").
					WillReturnRows(sqlmock.NewRows([]string{"external_id"}).
						AddRow("FIT001").
						AddRow("FIT003"))
			},
			want: map[string]bool{"FIT001": true, "FIT003": true},
		},
		{
			name:        "given empty input when finding then returns empty map without query",
			externalIDs: []string{},
			setupMock:   func(mock sqlmock.Sqlmock) {},
			want:        map[string]bool{},
		},
		{
			name:        "given db failure when finding then returns error",
			externalIDs: []string{"FIT001"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT external_id FROM statements WHERE account_id").
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

			result, findErr := repo.FindExternalIDs(context.Background(), accountID, tt.externalIDs)

			if tt.wantErr {
				assert.Error(t, findErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, findErr)
				assert.Equal(t, tt.want, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

package statement

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	accountdomain "github.com/DenysonJ/financial-wallet/internal/domain/account"
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	stmtdomain "github.com/DenysonJ/financial-wallet/internal/domain/statement"
	"github.com/DenysonJ/financial-wallet/internal/usecases/statement/dto"
	"github.com/DenysonJ/financial-wallet/pkg/ofx"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// validOFXContent returns a minimal valid SGML OFX with the given transactions.
func validOFXContent(transactions string) string {
	return `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:USASCII
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<SIGNONMSGSRSV1>
<SONRS>
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<DTSERVER>20260301
<LANGUAGE>POR
</SONRS>
</SIGNONMSGSRSV1>
<BANKMSGSRSV1>
<STMTTRNRS>
<TRNUID>1
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<STMTRS>
<CURDEF>BRL
<BANKACCTFROM>
<BANKID>001
<ACCTID>999
<ACCTTYPE>CHECKING
</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20260101
<DTEND>20260131
` + transactions + `
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`
}

const twoTransactions = `<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20260105
<TRNAMT>1000.00
<FITID>FIT001
<NAME>Salario
<MEMO>Pagamento
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20260110
<TRNAMT>-50.75
<FITID>FIT002
<MEMO>Uber
</STMTTRN>`

const emptyOFX = `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:USASCII
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<SIGNONMSGSRSV1>
<SONRS>
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<DTSERVER>20260301
<LANGUAGE>POR
</SONRS>
</SIGNONMSGSRSV1>
<BANKMSGSRSV1>
<STMTTRNRS>
<TRNUID>1
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<STMTRS>
<CURDEF>BRL
<BANKACCTFROM>
<BANKID>001
<ACCTID>999
<ACCTTYPE>CHECKING
</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20260101
<DTEND>20260131
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`

func TestImportUseCase_Execute(t *testing.T) {
	ownerID := vo.NewID()
	otherUserID := vo.NewID()
	accountID := vo.NewID()
	now := time.Now()

	activeAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: true, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}
	inactiveAccount := &accountdomain.Account{
		ID: accountID, UserID: ownerID, Name: "Nubank", Type: accountvo.TypeBankAccount,
		Active: false, Balance: 10000, CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name            string
		input           dto.ImportOFXInput
		accountResult   *accountdomain.Account
		accountErr      error
		existingIDs     map[string]bool
		findIDsErr      error
		batchBalance    int64
		batchErr        error
		wantErr         error
		wantErrMsg      string
		wantOutput      *dto.ImportOutput
		skipAccountCall bool
		skipFindIDs     bool
		skipBatchCall   bool
	}{
		{
			name: "given valid OFX file when importing then creates all statements",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountResult: activeAccount,
			existingIDs:   map[string]bool{},
			batchBalance:  110925,
			wantOutput:    &dto.ImportOutput{TotalTransactions: 2, Created: 2, Skipped: 0},
		},
		{
			name: "given OFX with duplicate FITIDs when importing then skips duplicates",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountResult: activeAccount,
			existingIDs:   map[string]bool{"FIT001": true},
			batchBalance:  94925,
			wantOutput:    &dto.ImportOutput{TotalTransactions: 2, Created: 1, Skipped: 1},
		},
		{
			name: "given all duplicate FITIDs when importing then skips all",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountResult: activeAccount,
			existingIDs:   map[string]bool{"FIT001": true, "FIT002": true},
			wantOutput:    &dto.ImportOutput{TotalTransactions: 2, Created: 0, Skipped: 2},
			skipBatchCall: true,
		},
		{
			name: "given invalid account ID when importing then returns invalid ID error",
			input: dto.ImportOFXInput{
				AccountID:        "invalid",
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			wantErr:         vo.ErrInvalidID,
			skipAccountCall: true,
			skipFindIDs:     true,
			skipBatchCall:   true,
		},
		{
			name: "given nonexistent account when importing then returns not found",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountErr:    accountdomain.ErrAccountNotFound,
			wantErr:       accountdomain.ErrAccountNotFound,
			skipFindIDs:   true,
			skipBatchCall: true,
		},
		{
			name: "given other users account when importing then returns not found",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: otherUserID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountResult: activeAccount,
			wantErr:       stmtdomain.ErrStatementNotFound,
			skipFindIDs:   true,
			skipBatchCall: true,
		},
		{
			name: "given inactive account when importing then returns not active",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountResult: inactiveAccount,
			wantErr:       stmtdomain.ErrAccountNotActive,
			skipFindIDs:   true,
			skipBatchCall: true,
		},
		{
			name: "given invalid OFX file when importing then returns parse error",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader("not a valid OFX file"),
			},
			accountResult: activeAccount,
			wantErr:       ofx.ErrInvalidFormat,
			skipFindIDs:   true,
			skipBatchCall: true,
		},
		{
			name: "given empty OFX file when importing then returns no transactions error",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(emptyOFX),
			},
			accountResult: activeAccount,
			wantErr:       ofx.ErrNoTransactions,
			skipFindIDs:   true,
			skipBatchCall: true,
		},
		{
			name: "given repo failure on batch create when importing then returns error",
			input: dto.ImportOFXInput{
				AccountID:        accountID.String(),
				RequestingUserID: ownerID.String(),
				FileContent:      strings.NewReader(validOFXContent(twoTransactions)),
			},
			accountResult: activeAccount,
			existingIDs:   map[string]bool{},
			batchErr:      errors.New("database error"),
			wantErrMsg:    "database error",
			skipBatchCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			mockAccRepo := &mockAccountRepository{}

			if !tt.skipAccountCall {
				mockAccRepo.On("FindByID", mock.Anything, mock.AnythingOfType("vo.ID")).
					Return(tt.accountResult, tt.accountErr)
			}
			if !tt.skipFindIDs {
				mockRepo.On("FindExternalIDs", mock.Anything, mock.AnythingOfType("vo.ID"), mock.AnythingOfType("[]string")).
					Return(tt.existingIDs, tt.findIDsErr)
			}
			if !tt.skipBatchCall {
				mockRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*statement.Statement"), mock.AnythingOfType("vo.ID")).
					Return(tt.batchBalance, tt.batchErr)
			}

			uc := NewImportUseCase(mockRepo, mockAccRepo)
			output, execErr := uc.Execute(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, execErr, tt.wantErr)
				assert.Nil(t, output)
			} else if tt.wantErrMsg != "" {
				assert.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.wantErrMsg)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, execErr)
				assert.NotNil(t, output)
				assert.Equal(t, tt.wantOutput.TotalTransactions, output.TotalTransactions)
				assert.Equal(t, tt.wantOutput.Created, output.Created)
				assert.Equal(t, tt.wantOutput.Skipped, output.Skipped)
			}

			if tt.skipAccountCall {
				mockAccRepo.AssertNotCalled(t, "FindByID")
			}
			if tt.skipFindIDs {
				mockRepo.AssertNotCalled(t, "FindExternalIDs")
			}
			if tt.skipBatchCall {
				mockRepo.AssertNotCalled(t, "CreateBatch")
			}
		})
	}
}

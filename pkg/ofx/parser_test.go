package ofx

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name         string
		fixturePath  string
		wantTxnCount int
		wantErr      error
		validate     func(t *testing.T, result *ParseResult)
	}{
		{
			name:         "given valid SGML v1 file when parsing then extracts all transactions",
			fixturePath:  "testdata/valid_v1.ofx",
			wantTxnCount: 3,
			validate: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, "102", result.Header.Version)

				// First transaction: credit 5000.00
				txn0 := result.Transactions[0]
				assert.Equal(t, "202601050001", txn0.FITID)
				assert.Equal(t, "CREDIT", txn0.Type)
				assert.Equal(t, int64(500000), txn0.Amount)
				assert.Equal(t, "Salario", txn0.Name)
				assert.Equal(t, "Pagamento mensal", txn0.Memo)
				assert.True(t, txn0.DatePosted.Equal(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)))

				// Second transaction: debit -150.75
				txn1 := result.Transactions[1]
				assert.Equal(t, "202601100002", txn1.FITID)
				assert.Equal(t, "DEBIT", txn1.Type)
				assert.Equal(t, int64(-15075), txn1.Amount)
				assert.Equal(t, "Supermercado", txn1.Name)
				assert.Equal(t, "Compras do mes", txn1.Memo)
				assert.True(t, txn1.DatePosted.Equal(time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)))

				// Third transaction: debit -30.50 (no NAME, only MEMO)
				txn2 := result.Transactions[2]
				assert.Equal(t, "202601150003", txn2.FITID)
				assert.Equal(t, "DEBIT", txn2.Type)
				assert.Equal(t, int64(-3050), txn2.Amount)
				assert.Equal(t, "", txn2.Name)
				assert.Equal(t, "Uber", txn2.Memo)
			},
		},
		{
			name:         "given valid XML v2 file when parsing then extracts all transactions",
			fixturePath:  "testdata/valid_v2.ofx",
			wantTxnCount: 2,
			validate: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, "200", result.Header.Version)

				// First transaction: credit 3200.00
				txn0 := result.Transactions[0]
				assert.Equal(t, "V2TXN001", txn0.FITID)
				assert.Equal(t, "CREDIT", txn0.Type)
				assert.Equal(t, int64(320000), txn0.Amount)
				assert.Equal(t, "Freelance", txn0.Name)
				assert.Equal(t, "Projeto web", txn0.Memo)

				// Second transaction: debit -89.90
				txn1 := result.Transactions[1]
				assert.Equal(t, "V2TXN002", txn1.FITID)
				assert.Equal(t, "DEBIT", txn1.Type)
				assert.Equal(t, int64(-8990), txn1.Amount)
				assert.Equal(t, "Netflix", txn1.Name)
			},
		},
		{
			name:        "given OFX with no transactions when parsing then returns no transactions error",
			fixturePath: "testdata/empty.ofx",
			wantErr:     ErrNoTransactions,
		},
		{
			name:        "given invalid file when parsing then returns invalid format error",
			fixturePath: "testdata/invalid.ofx",
			wantErr:     ErrInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, openErr := os.Open(tt.fixturePath)
			require.NoError(t, openErr)
			defer f.Close()

			result, parseErr := Parse(f)

			if tt.wantErr != nil {
				assert.ErrorIs(t, parseErr, tt.wantErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, parseErr)
			require.NotNil(t, result)
			assert.Len(t, result.Transactions, tt.wantTxnCount)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestParse_EmptyInput(t *testing.T) {
	_, parseErr := Parse(strings.NewReader(""))
	assert.ErrorIs(t, parseErr, ErrInvalidFormat)
}

func TestParse_NoOFXTag(t *testing.T) {
	_, parseErr := Parse(strings.NewReader("OFXHEADER:100\nDATA:OFXSGML\nVERSION:102\n"))
	assert.ErrorIs(t, parseErr, ErrInvalidFormat)
}

func TestParse_InlineString_SGML(t *testing.T) {
	ofxContent := `OFXHEADER:100
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
<DTSTART>20260301
<DTEND>20260301
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20260301
<TRNAMT>100.00
<FITID>INLINE001
<NAME>Test Credit
<MEMO>Inline test
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`

	result, parseErr := Parse(strings.NewReader(ofxContent))
	require.NoError(t, parseErr)
	require.Len(t, result.Transactions, 1)

	txn := result.Transactions[0]
	assert.Equal(t, "INLINE001", txn.FITID)
	assert.Equal(t, "CREDIT", txn.Type)
	assert.Equal(t, int64(10000), txn.Amount)
	assert.Equal(t, "Test Credit", txn.Name)
	assert.Equal(t, "Inline test", txn.Memo)
}

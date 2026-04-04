package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccountType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AccountType
		wantErr error
	}{
		{name: "bank_account válido", input: "bank_account", want: TypeBankAccount},
		{name: "credit_card válido", input: "credit_card", want: TypeCreditCard},
		{name: "cash válido", input: "cash", want: TypeCash},
		{name: "string vazia", input: "", wantErr: ErrInvalidAccountType},
		{name: "tipo desconhecido", input: "savings", wantErr: ErrInvalidAccountType},
		{name: "uppercase", input: "BANK_ACCOUNT", wantErr: ErrInvalidAccountType},
		{name: "com espaço", input: " cash", wantErr: ErrInvalidAccountType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, typeErr := NewAccountType(tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, typeErr, tt.wantErr)
				assert.Equal(t, AccountType(""), got)
			} else {
				assert.NoError(t, typeErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseAccountType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  AccountType
	}{
		{name: "tipo válido", input: "bank_account", want: TypeBankAccount},
		{name: "tipo inválido aceito sem validação", input: "unknown", want: AccountType("unknown")},
		{name: "string vazia", input: "", want: AccountType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAccountType(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAccountType_String(t *testing.T) {
	tests := []struct {
		name string
		at   AccountType
		want string
	}{
		{name: "bank_account", at: TypeBankAccount, want: "bank_account"},
		{name: "credit_card", at: TypeCreditCard, want: "credit_card"},
		{name: "cash", at: TypeCash, want: "cash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.at.String())
		})
	}
}

func TestAccountType_Value(t *testing.T) {
	tests := []struct {
		name string
		at   AccountType
		want string
	}{
		{name: "bank_account", at: TypeBankAccount, want: "bank_account"},
		{name: "credit_card", at: TypeCreditCard, want: "credit_card"},
		{name: "cash", at: TypeCash, want: "cash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, valErr := tt.at.Value()
			assert.NoError(t, valErr)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestAccountType_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    AccountType
		wantErr bool
		errMsg  string
	}{
		{name: "string válida", input: "bank_account", want: TypeBankAccount},
		{name: "[]byte válido", input: []byte("credit_card"), want: TypeCreditCard},
		{name: "nil retorna erro", input: nil, wantErr: true, errMsg: "account type cannot be nil"},
		{name: "tipo inválido (int)", input: 123, wantErr: true, errMsg: "invalid type for AccountType"},
		{name: "tipo inválido (bool)", input: true, wantErr: true, errMsg: "invalid type for AccountType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var at AccountType
			scanErr := at.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, scanErr)
				assert.Contains(t, scanErr.Error(), tt.errMsg)
			} else {
				assert.NoError(t, scanErr)
				assert.Equal(t, tt.want, at)
			}
		})
	}
}

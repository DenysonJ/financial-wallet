package account

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cents(v int64) *int64 { return &v }

func TestNewAccount_CreditLimitMatrix(t *testing.T) {
	tests := []struct {
		name        string
		accountType vo.AccountType
		limit       *int64
		wantErr     error
	}{
		{
			name:        "GIVEN a credit card WHEN created with a valid limit THEN accepts it (INV-01)",
			accountType: vo.TypeCreditCard,
			limit:       cents(500000),
		},
		{
			name:        "GIVEN a credit card WHEN created without a limit THEN requires one (RN-01, INV-01)",
			accountType: vo.TypeCreditCard,
			limit:       nil,
			wantErr:     ErrCreditLimitRequired,
		},
		{
			name:        "GIVEN a credit card WHEN created with a zero limit THEN rejects the value (RN-03, INV-03)",
			accountType: vo.TypeCreditCard,
			limit:       cents(0),
			wantErr:     vo.ErrInvalidCreditLimit,
		},
		{
			name:        "GIVEN a credit card WHEN created above the ceiling THEN rejects the value (RN-03, INV-03)",
			accountType: vo.TypeCreditCard,
			limit:       cents(vo.MaxCreditLimit + 1),
			wantErr:     vo.ErrInvalidCreditLimit,
		},
		{
			name:        "GIVEN a bank account WHEN created without a limit THEN accepts it (INV-02)",
			accountType: vo.TypeBankAccount,
			limit:       nil,
		},
		{
			name:        "GIVEN a bank account WHEN created with a limit THEN rejects it (RN-02, INV-02)",
			accountType: vo.TypeBankAccount,
			limit:       cents(100000),
			wantErr:     ErrCreditLimitNotAllowed,
		},
		{
			// A ordem das invariantes importa: tipo × presença antes do valor.
			name:        "GIVEN a bank account WHEN created with a zero limit THEN rejects by type, not by value (RN-02)",
			accountType: vo.TypeBankAccount,
			limit:       cents(0),
			wantErr:     ErrCreditLimitNotAllowed,
		},
		{
			name:        "GIVEN a cash account WHEN created without a limit THEN accepts it (INV-02)",
			accountType: vo.TypeCash,
			limit:       nil,
		},
		{
			name:        "GIVEN a cash account WHEN created with a limit THEN rejects it (RN-02, INV-02)",
			accountType: vo.TypeCash,
			limit:       cents(50000),
			wantErr:     ErrCreditLimitNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, newErr := NewAccount(uservo.NewID(), "Conta", tt.accountType, "", tt.limit)

			if tt.wantErr != nil {
				assert.ErrorIs(t, newErr, tt.wantErr)
				assert.Nil(t, a, "entidade inválida nunca deve ser construída (INV-05)")
				return
			}

			require.NoError(t, newErr)
			require.NotNil(t, a)
			if tt.limit == nil {
				assert.Nil(t, a.CreditLimit)
				return
			}
			require.NotNil(t, a.CreditLimit)
			assert.Equal(t, *tt.limit, a.CreditLimit.Int64())
		})
	}
}

func TestNewAccount_CreditCardStartsWithZeroBalance(t *testing.T) {
	a, newErr := NewAccount(uservo.NewID(), "Cartão", vo.TypeCreditCard, "", cents(500000))

	require.NoError(t, newErr)
	assert.Equal(t, int64(0), a.Balance)
	assert.Equal(t, int64(500000), a.CreditLimit.Int64())
}

func TestAccount_SetCreditLimit(t *testing.T) {
	tests := []struct {
		name        string
		accountType vo.AccountType
		initial     *int64
		newLimit    int64
		wantErr     error
	}{
		{
			name:        "GIVEN a credit card WHEN setting a new limit THEN replaces the previous one (RN-07)",
			accountType: vo.TypeCreditCard,
			initial:     cents(500000),
			newLimit:    800000,
		},
		{
			name:        "GIVEN a credit card with an open invoice WHEN reducing below the amount used THEN allows it (RN-11)",
			accountType: vo.TypeCreditCard,
			initial:     cents(500000),
			newLimit:    100000,
		},
		{
			name:        "GIVEN a credit card WHEN setting a zero limit THEN rejects the value (RN-03, INV-03)",
			accountType: vo.TypeCreditCard,
			initial:     cents(500000),
			newLimit:    0,
			wantErr:     vo.ErrInvalidCreditLimit,
		},
		{
			name:        "GIVEN a cash account WHEN setting a limit THEN rejects it (RN-08, INV-02)",
			accountType: vo.TypeCash,
			initial:     nil,
			newLimit:    50000,
			wantErr:     ErrCreditLimitNotAllowed,
		},
		{
			name:        "GIVEN a bank account WHEN setting a limit THEN rejects it (RN-08, INV-02)",
			accountType: vo.TypeBankAccount,
			initial:     nil,
			newLimit:    50000,
			wantErr:     ErrCreditLimitNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, newErr := NewAccount(uservo.NewID(), "Conta", tt.accountType, "", tt.initial)
			require.NoError(t, newErr)
			// Saldo devedor: a conta já gastou mais do que o novo limite.
			a.Balance = -300000
			before := a.UpdatedAt

			limitErr := a.SetCreditLimit(tt.newLimit)

			if tt.wantErr != nil {
				assert.ErrorIs(t, limitErr, tt.wantErr)
				if tt.initial == nil {
					assert.Nil(t, a.CreditLimit, "limite recusado não deve ser atribuído")
				} else {
					assert.Equal(t, *tt.initial, a.CreditLimit.Int64(), "limite recusado não deve sobrescrever o anterior")
				}
				return
			}

			require.NoError(t, limitErr)
			assert.Equal(t, tt.newLimit, a.CreditLimit.Int64())
			//mexe em updated_at, nunca no saldo.
			assert.GreaterOrEqual(t, a.UpdatedAt.UnixNano(), before.UnixNano())
			assert.Equal(t, int64(-300000), a.Balance)
		})
	}
}

func TestAcceptsCreditLimit(t *testing.T) {
	assert.True(t, AcceptsCreditLimit(vo.TypeCreditCard))
	assert.False(t, AcceptsCreditLimit(vo.TypeBankAccount))
	assert.False(t, AcceptsCreditLimit(vo.TypeCash))
}

package account

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func TestNewAccount(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")

	a := NewAccount(userID, "Nubank", accType, "Conta corrente")

	assert.NotEmpty(t, a.ID)
	assert.Equal(t, userID, a.UserID)
	assert.Equal(t, "Nubank", a.Name)
	assert.Equal(t, vo.TypeBankAccount, a.Type)
	assert.Equal(t, "Conta corrente", a.Description)
	assert.True(t, a.Active)
	assert.NotZero(t, a.CreatedAt)
	assert.NotZero(t, a.UpdatedAt)
}

func TestAccount_Deactivate(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("cash")
	a := NewAccount(userID, "Caixa", accType, "")

	a.Deactivate()

	assert.False(t, a.Active)
}

func TestAccount_UpdateName(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("credit_card")
	a := NewAccount(userID, "Old Card", accType, "")
	oldUpdatedAt := a.UpdatedAt

	a.UpdateName("New Card")

	assert.Equal(t, "New Card", a.Name)
	assert.GreaterOrEqual(t, a.UpdatedAt.UnixNano(), oldUpdatedAt.UnixNano())
}

func TestAccount_UpdateDescription(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")
	a := NewAccount(userID, "Nubank", accType, "")
	oldUpdatedAt := a.UpdatedAt

	a.UpdateDescription("Conta corrente principal")

	assert.Equal(t, "Conta corrente principal", a.Description)
	assert.GreaterOrEqual(t, a.UpdatedAt.UnixNano(), oldUpdatedAt.UnixNano())
}

func TestNewAccount_BalanceInitializedToZero(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")

	a := NewAccount(userID, "Nubank", accType, "")

	assert.Equal(t, int64(0), a.Balance)
}

func TestAccount_CreditBalance(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")
	a := NewAccount(userID, "Nubank", accType, "")

	a.CreditBalance(5000)

	assert.Equal(t, int64(5000), a.Balance)

	a.CreditBalance(3000)

	assert.Equal(t, int64(8000), a.Balance)
}

func TestAccount_DebitBalance(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")
	a := NewAccount(userID, "Nubank", accType, "")
	a.CreditBalance(10000)

	a.DebitBalance(4000)

	assert.Equal(t, int64(6000), a.Balance)
}

func TestAccount_DebitBalance_GoesNegative(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")
	a := NewAccount(userID, "Nubank", accType, "")
	a.CreditBalance(1000)

	a.DebitBalance(5000)

	assert.Equal(t, int64(-4000), a.Balance)
}

func TestAccount_DebitBalance_ExactBalance(t *testing.T) {
	userID := uservo.NewID()
	accType, _ := vo.NewAccountType("bank_account")
	a := NewAccount(userID, "Nubank", accType, "")
	a.CreditBalance(2500)

	a.DebitBalance(2500)

	assert.Equal(t, int64(0), a.Balance)
}

func TestAccountType_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  vo.AccountType
	}{
		{"bank_account", "bank_account", vo.TypeBankAccount},
		{"credit_card", "credit_card", vo.TypeCreditCard},
		{"cash", "cash", vo.TypeCash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, typeErr := vo.NewAccountType(tt.input)
			assert.NoError(t, typeErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAccountType_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"unknown type", "savings"},
		{"uppercase", "BANK_ACCOUNT"},
		{"typo", "credit-card"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, typeErr := vo.NewAccountType(tt.input)
			assert.ErrorIs(t, typeErr, vo.ErrInvalidAccountType)
		})
	}
}

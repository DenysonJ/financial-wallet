package account

import (
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Account é a Entidade principal (Aggregate Root) do domínio account.
// Representa um container financeiro (conta bancária, cartão de crédito, caixa).
type Account struct {
	ID          uservo.ID
	UserID      uservo.ID
	Name        string
	Type        vo.AccountType
	Description string
	Balance     int64
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewAccount cria um Account com valores padrão.
func NewAccount(userID uservo.ID, name string, accountType vo.AccountType, description string) *Account {
	return &Account{
		ID:          uservo.NewID(),
		UserID:      userID,
		Name:        name,
		Type:        accountType,
		Description: description,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// Deactivate desativa a account (soft delete).
func (a *Account) Deactivate() {
	a.Active = false
	a.UpdatedAt = time.Now()
}

// UpdateName atualiza o nome da account.
func (a *Account) UpdateName(name string) {
	a.Name = name
	a.UpdatedAt = time.Now()
}

// UpdateDescription atualiza a descrição da account.
func (a *Account) UpdateDescription(description string) {
	a.Description = description
	a.UpdatedAt = time.Now()
}

// CreditBalance increases the account balance by the given amount.
func (a *Account) CreditBalance(amount int64) {
	a.Balance += amount
	a.UpdatedAt = time.Now()
}

// DebitBalance decreases the account balance by the given amount.
// Balance can go negative (equivalent to owing money).
func (a *Account) DebitBalance(amount int64) {
	a.Balance -= amount
	a.UpdatedAt = time.Now()
}

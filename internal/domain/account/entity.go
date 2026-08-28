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
	// CreditLimit é o limite de crédito da account, em centavos. Nil significa
	// ausência de limite — estado válido apenas para tipos diferentes de credit_card
	CreditLimit *vo.CreditLimit
}

// AcceptsCreditLimit informa se o tipo de account admite limite de crédito.
func AcceptsCreditLimit(accountType vo.AccountType) bool {
	return accountType == vo.TypeCreditCard
}

// NewAccount cria um Account com valores padrão. creditLimitCents é o limite em
// centavos vindo da borda: obrigatório para credit_card, proibido nos demais tipos
func NewAccount(
	userID uservo.ID,
	name string,
	accountType vo.AccountType,
	description string,
	creditLimitCents *int64,
) (*Account, error) {
	creditLimit, limitErr := resolveCreditLimit(accountType, creditLimitCents)
	if limitErr != nil {
		return nil, limitErr
	}

	return &Account{
		ID:          uservo.NewID(),
		UserID:      userID,
		Name:        name,
		Type:        accountType,
		Description: description,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreditLimit: creditLimit,
	}, nil
}

func (a *Account) SetCreditLimit(creditLimitCents int64) error {
	if !AcceptsCreditLimit(a.Type) {
		return ErrCreditLimitNotAllowed
	}

	limit, limitErr := vo.NewCreditLimit(creditLimitCents)
	if limitErr != nil {
		return limitErr
	}

	a.CreditLimit = &limit
	a.UpdatedAt = time.Now()
	return nil
}

func resolveCreditLimit(accountType vo.AccountType, cents *int64) (*vo.CreditLimit, error) {
	if !AcceptsCreditLimit(accountType) {
		if cents != nil {
			return nil, ErrCreditLimitNotAllowed
		}
		return nil, nil
	}

	if cents == nil {
		return nil, ErrCreditLimitRequired
	}

	limit, limitErr := vo.NewCreditLimit(*cents)
	if limitErr != nil {
		return nil, limitErr
	}
	return &limit, nil
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

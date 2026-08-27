package account

import (
	accountvo "github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
)

// creditLimitCents converte o limite de crédito do domínio no ponteiro de
// centavos exposto nos DTOs. Retorna nil quando a account não tem limite, para
// que o campo apareça como null na resposta.
func creditLimitCents(limit *accountvo.CreditLimit) *int64 {
	if limit == nil {
		return nil
	}
	cents := limit.Int64()
	return &cents
}

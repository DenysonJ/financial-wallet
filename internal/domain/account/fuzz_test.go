package account

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/account/vo"
	uservo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func FuzzNewAccount(f *testing.F) {
	f.Add("Nubank", "Conta corrente")
	f.Add("", "")
	f.Add("a", "b")
	f.Add(string(make([]byte, 10000)), string(make([]byte, 10000)))
	f.Add("Name\x00null", "Desc\x00null")
	f.Add("名前", "説明")
	f.Add("Name\nwith\nnewlines", "Desc\twith\ttabs")

	f.Fuzz(func(t *testing.T, name, description string) {
		userID := uservo.NewID()
		a := NewAccount(userID, name, vo.TypeBankAccount, description)

		// Must never panic
		assert.NotNil(t, a)
		assert.Equal(t, name, a.Name)
		assert.Equal(t, description, a.Description)
		assert.Equal(t, userID, a.UserID)
		assert.True(t, a.Active)
		assert.NotEmpty(t, a.ID)
	})
}

func FuzzAccountUpdateName(f *testing.F) {
	f.Add("New Name")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("Name\x00with\x00nulls")
	f.Add("名前更新")

	f.Fuzz(func(t *testing.T, name string) {
		a := NewAccount(uservo.NewID(), "Original", vo.TypeCash, "")
		a.UpdateName(name)

		assert.Equal(t, name, a.Name)
	})
}

func FuzzAccountUpdateDescription(f *testing.F) {
	f.Add("New Description")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("Desc\nwith\nnewlines\tand\ttabs")

	f.Fuzz(func(t *testing.T, description string) {
		a := NewAccount(uservo.NewID(), "Test", vo.TypeCreditCard, "Original")
		a.UpdateDescription(description)

		assert.Equal(t, description, a.Description)
	})
}

package category

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func FuzzNewCategory(f *testing.F) {
	f.Add("Mercado")
	f.Add("")
	f.Add("a")
	f.Add(string(make([]byte, 10000)))
	f.Add("Name\x00null")
	f.Add("Categoria com nome muito longo e caracteres unicode 名前")
	f.Add("Name\nwith\nnewlines\tand\ttabs")

	f.Fuzz(func(t *testing.T, name string) {
		userID := pkgvo.NewID()
		c := NewCategory(userID, name, vo.TypeDebit)

		assert.NotNil(t, c)
		assert.Equal(t, name, c.Name)
		assert.NotEmpty(t, c.ID)
		assert.False(t, c.IsSystem())
	})
}

func FuzzNewSystemCategory(f *testing.F) {
	f.Add("Salário")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("名前")

	f.Fuzz(func(t *testing.T, name string) {
		c := NewSystemCategory(name, vo.TypeCredit)

		assert.NotNil(t, c)
		assert.Equal(t, name, c.Name)
		assert.True(t, c.IsSystem())
		assert.Nil(t, c.UserID)
	})
}

func FuzzCategoryRename(f *testing.F) {
	f.Add("New Name")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("Name\x00with\x00nulls")

	f.Fuzz(func(t *testing.T, name string) {
		c := NewCategory(pkgvo.NewID(), "Original", vo.TypeDebit)
		originalType := c.Type
		_ = c.Rename(name)

		assert.Equal(t, name, c.Name)
		assert.Equal(t, originalType, c.Type)
	})
}

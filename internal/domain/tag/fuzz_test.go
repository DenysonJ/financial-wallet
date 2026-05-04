package tag

import (
	"testing"

	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/stretchr/testify/assert"
)

func FuzzNewTag(f *testing.F) {
	f.Add("viagem-2026")
	f.Add("")
	f.Add("a")
	f.Add(string(make([]byte, 10000)))
	f.Add("Name\x00null")
	f.Add("with-hyphen-and-numbers-2026")
	f.Add("Name\nwith\nnewlines\tand\ttabs")

	f.Fuzz(func(t *testing.T, name string) {
		userID := pkgvo.NewID()
		tag := NewTag(userID, name)

		assert.NotNil(t, tag)
		assert.Equal(t, name, tag.Name)
		assert.NotEmpty(t, tag.ID)
		assert.False(t, tag.IsSystem())
	})
}

func FuzzNewSystemTag(f *testing.F) {
	f.Add("recorrente")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("名前")

	f.Fuzz(func(t *testing.T, name string) {
		tag := NewSystemTag(name)

		assert.NotNil(t, tag)
		assert.Equal(t, name, tag.Name)
		assert.True(t, tag.IsSystem())
		assert.Nil(t, tag.UserID)
	})
}

func FuzzTagRename(f *testing.F) {
	f.Add("New Name")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("Name\x00with\x00nulls")

	f.Fuzz(func(t *testing.T, name string) {
		tag := NewTag(pkgvo.NewID(), "Original")
		_ = tag.Rename(name)

		assert.Equal(t, name, tag.Name)
	})
}

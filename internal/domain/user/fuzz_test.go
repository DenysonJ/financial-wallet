package user

import (
	"testing"

	"github.com/DenysonJ/financial-wallet/internal/domain/user/vo"
	"github.com/stretchr/testify/assert"
)

func FuzzNewUser(f *testing.F) {
	f.Add("João Silva")
	f.Add("")
	f.Add("a")
	f.Add(string(make([]byte, 10000)))
	f.Add("Name\x00null")
	f.Add("名前太郎")
	f.Add("Name\nwith\nnewlines")

	f.Fuzz(func(t *testing.T, name string) {
		email := vo.ParseEmail("test@example.com")
		u := NewUser(name, email)

		// Must never panic
		assert.NotNil(t, u)
		assert.Equal(t, name, u.Name)
		assert.True(t, u.Active)
		assert.NotEmpty(t, u.ID)
	})
}

func FuzzUserUpdateName(f *testing.F) {
	f.Add("New Name")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("名前更新")
	f.Add("Name\x00with\x00nulls")

	f.Fuzz(func(t *testing.T, name string) {
		email := vo.ParseEmail("test@example.com")
		u := NewUser("Original", email)
		u.UpdateName(name)

		assert.Equal(t, name, u.Name)
	})
}

package role

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzNewRole(f *testing.F) {
	f.Add("admin", "Full access")
	f.Add("", "")
	f.Add("a", "b")
	f.Add(string(make([]byte, 10000)), string(make([]byte, 10000)))
	f.Add("role\x00null", "desc\x00null")
	f.Add("管理者", "全アクセス")

	f.Fuzz(func(t *testing.T, name, description string) {
		r := NewRole(name, description)

		// Must never panic
		assert.NotNil(t, r)
		assert.Equal(t, name, r.Name)
		assert.Equal(t, description, r.Description)
		assert.NotEmpty(t, r.ID)
	})
}

func FuzzRoleUpdateName(f *testing.F) {
	f.Add("New Role")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("ロール名")

	f.Fuzz(func(t *testing.T, name string) {
		r := NewRole("Original", "Desc")
		r.UpdateName(name)

		assert.Equal(t, name, r.Name)
	})
}

func FuzzRoleUpdateDescription(f *testing.F) {
	f.Add("New Description")
	f.Add("")
	f.Add(string(make([]byte, 10000)))
	f.Add("Desc\nwith\nnewlines")

	f.Fuzz(func(t *testing.T, description string) {
		r := NewRole("Test", "Original")
		r.UpdateDescription(description)

		assert.Equal(t, description, r.Description)
	})
}

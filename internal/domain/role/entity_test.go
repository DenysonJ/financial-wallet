package role

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRole(t *testing.T) {
	r := NewRole("admin", "Full access to all resources")

	assert.NotEmpty(t, r.ID)
	assert.Equal(t, "admin", r.Name)
	assert.Equal(t, "Full access to all resources", r.Description)
	assert.NotZero(t, r.CreatedAt)
	assert.NotZero(t, r.UpdatedAt)
}

func TestRole_UpdateDescription(t *testing.T) {
	r := NewRole("editor", "Can edit content")
	originalUpdatedAt := r.UpdatedAt

	r.UpdateDescription("Can edit and publish content")

	assert.Equal(t, "Can edit and publish content", r.Description)
	assert.GreaterOrEqual(t, r.UpdatedAt.UnixNano(), originalUpdatedAt.UnixNano())
}

func TestRole_UpdateName(t *testing.T) {
	r := NewRole("viewer", "Read-only access")
	originalUpdatedAt := r.UpdatedAt

	r.UpdateName("reader")

	assert.Equal(t, "reader", r.Name)
	assert.GreaterOrEqual(t, r.UpdatedAt.UnixNano(), originalUpdatedAt.UnixNano())
}

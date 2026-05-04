package category

import (
	"time"

	"github.com/DenysonJ/financial-wallet/internal/domain/category/vo"
	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// Category is the entity of the category domain.
//
// UserID is a pointer because system defaults have user_id NULL — nil signals
// "scope = system".
type Category struct {
	ID        pkgvo.ID
	UserID    *pkgvo.ID
	Name      string
	Type      vo.CategoryType
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCategory creates a user-scoped custom category.
func NewCategory(userID pkgvo.ID, name string, categoryType vo.CategoryType) *Category {
	now := time.Now()
	return &Category{
		ID:        pkgvo.NewID(),
		UserID:    &userID,
		Name:      name,
		Type:      categoryType,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewSystemCategory creates a default category visible to all users.
// Used only in seeds; not exposed via API.
func NewSystemCategory(name string, categoryType vo.CategoryType) *Category {
	now := time.Now()
	return &Category{
		ID:        pkgvo.NewID(),
		UserID:    nil,
		Name:      name,
		Type:      categoryType,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsSystem reports whether the category is a system default (read-only).
func (c *Category) IsSystem() bool {
	return c.UserID == nil
}

// Rename updates Name and UpdatedAt; Type is immutable after creation.
func (c *Category) Rename(name string) error {
	if c.IsSystem() {
		return ErrCategoryReadOnly
	}
	c.Name = name
	c.UpdatedAt = time.Now()
	return nil
}

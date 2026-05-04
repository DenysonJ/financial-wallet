package tag

import (
	"time"

	pkgvo "github.com/DenysonJ/financial-wallet/pkg/vo"
)

// MaxTagsPerStatement is the cap on unique tags per statement, enforced after dedup.
const MaxTagsPerStatement = 10

// Tag is the entity of the tag domain.
//
// UserID is a pointer because system defaults have user_id NULL — nil signals
// "scope = system".
type Tag struct {
	ID        pkgvo.ID
	UserID    *pkgvo.ID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTag creates a user-scoped custom tag.
func NewTag(userID pkgvo.ID, name string) *Tag {
	now := time.Now()
	return &Tag{
		ID:        pkgvo.NewID(),
		UserID:    &userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewSystemTag creates a default tag visible to all users.
// Used only in seeds; not exposed via API.
func NewSystemTag(name string) *Tag {
	now := time.Now()
	return &Tag{
		ID:        pkgvo.NewID(),
		UserID:    nil,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsSystem reports whether the tag is a system default (read-only).
func (t *Tag) IsSystem() bool {
	return t.UserID == nil
}

// Rename updates Name and UpdatedAt.
func (t *Tag) Rename(name string) error {
	if t.IsSystem() {
		return ErrTagReadOnly
	}
	t.Name = name
	t.UpdatedAt = time.Now()
	return nil
}

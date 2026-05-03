package dto

// CreateInput — body for POST /tags.
type CreateInput struct {
	UserID string `json:"-"` // injected by the handler from the authenticated context
	Name   string `json:"name" binding:"required,max=40"`
}

// UpdateInput — body for PATCH /tags/:id.
type UpdateInput struct {
	UserID string `json:"-"`
	ID     string `json:"-"`
	Name   string `json:"name" binding:"required,max=40"`
}

// DeleteInput — DELETE /tags/:id.
type DeleteInput struct {
	UserID string `json:"-"`
	ID     string `json:"-"`
}

// ListInput — GET /tags?scope=.
type ListInput struct {
	UserID string `json:"-"`
	Scope  string `form:"scope" binding:"omitempty,oneof=system user"`
}

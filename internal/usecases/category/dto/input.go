package dto

// CreateInput — body for POST /categories.
type CreateInput struct {
	UserID string `json:"-"` // injected by the handler from the authenticated context
	Name   string `json:"name" binding:"required,max=60"`
	Type   string `json:"type" binding:"required,oneof=credit debit"`
}

// UpdateInput — body for PATCH /categories/:id. Only `name`; `type` is immutable.
type UpdateInput struct {
	UserID string `json:"-"`
	ID     string `json:"-"`
	Name   string `json:"name" binding:"required,max=60"`
}

// DeleteInput — DELETE /categories/:id.
type DeleteInput struct {
	UserID string `json:"-"`
	ID     string `json:"-"`
}

// ListInput — GET /categories?type=&scope=.
type ListInput struct {
	UserID string `json:"-"`
	Type   string `form:"type" binding:"omitempty,oneof=credit debit"`
	Scope  string `form:"scope" binding:"omitempty,oneof=system user"`
}

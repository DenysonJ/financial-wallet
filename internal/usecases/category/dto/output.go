package dto

// CategoryOutput — canonical Category representation in the API.
type CategoryOutput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Scope     string `json:"scope"` // "system" or "user"
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateOutput — response body for POST /categories.
type CreateOutput struct {
	CategoryOutput
}

// UpdateOutput — response body for PATCH /categories/:id.
type UpdateOutput struct {
	CategoryOutput
}

// DeleteOutput — response body for DELETE /categories/:id.
type DeleteOutput struct {
	ID        string `json:"id"`
	DeletedAt string `json:"deleted_at"`
}

// ListOutput — response body for GET /categories.
type ListOutput struct {
	Data []CategoryOutput `json:"data"`
}

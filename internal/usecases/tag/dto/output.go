package dto

// TagOutput — canonical Tag representation in the API.
type TagOutput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Scope     string `json:"scope"` // "system" or "user"
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateOutput — response body for POST /tags.
type CreateOutput struct {
	TagOutput
}

// UpdateOutput — response body for PATCH /tags/:id.
type UpdateOutput struct {
	TagOutput
}

// DeleteOutput — response body for DELETE /tags/:id.
type DeleteOutput struct {
	ID        string `json:"id"`
	DeletedAt string `json:"deleted_at"`
}

// ListOutput — response body for GET /tags.
type ListOutput struct {
	Data []TagOutput `json:"data"`
}

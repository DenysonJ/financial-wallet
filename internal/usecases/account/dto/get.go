package dto

// =============================================================================
// Get Account DTOs
// =============================================================================

// GetInput representa os dados de entrada para buscar uma account.
type GetInput struct {
	ID               string `json:"id"` // UUID v7 da account
	RequestingUserID string `json:"-"`  // JWT user_id (empty = skip ownership check, e.g. admin/service-key)
}

// GetOutput representa os dados de saída da account encontrada.
type GetOutput struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Balance     int64  `json:"balance"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

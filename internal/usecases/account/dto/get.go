package dto

// =============================================================================
// Get Account DTOs
// =============================================================================

// GetInput representa os dados de entrada para buscar uma account.
type GetInput struct {
	ID string `json:"id"` // UUID v7 da account
}

// GetOutput representa os dados de saída da account encontrada.
type GetOutput struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

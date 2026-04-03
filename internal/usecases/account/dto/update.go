package dto

// =============================================================================
// Update Account DTOs
// =============================================================================

// UpdateInput representa os dados de entrada para atualizar uma account.
type UpdateInput struct {
	ID          string  `json:"-"`                                                  // ID vem da URL
	Name        *string `json:"name,omitempty" binding:"omitempty,max=255"`         // Nome (opcional)
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000"` // Descrição (opcional)
}

// UpdateOutput representa os dados de saída após atualização.
type UpdateOutput struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	UpdatedAt   string `json:"updated_at"`
}

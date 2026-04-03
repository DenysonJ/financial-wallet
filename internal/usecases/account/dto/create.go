package dto

// =============================================================================
// Create Account DTOs
// =============================================================================

// CreateInput representa os dados de entrada para criação de account.
type CreateInput struct {
	UserID      string `json:"-"`                                                  // Vem do JWT context
	Name        string `json:"name" binding:"required,max=255"`                    // Nome da conta
	Type        string `json:"type" binding:"required,max=50"`                     // Tipo: bank_account, credit_card, cash
	Description string `json:"description,omitempty" binding:"omitempty,max=1000"` // Descrição opcional
}

// CreateOutput representa os dados de saída após criação.
type CreateOutput struct {
	ID        string `json:"id"`         // ID gerado (UUID v7)
	CreatedAt string `json:"created_at"` // Timestamp no formato RFC3339
}

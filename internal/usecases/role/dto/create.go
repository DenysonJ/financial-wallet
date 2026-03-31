package dto

// =============================================================================
// Create Role DTOs
// =============================================================================

// CreateInput representa os dados de entrada para criacao de role.
type CreateInput struct {
	Name        string `json:"name" binding:"required,max=100"` // Nome da role
	Description string `json:"description" binding:"max=500"`   // Descricao da role
}

// CreateOutput representa os dados de saida apos criacao.
type CreateOutput struct {
	ID        string `json:"id"`         // ID gerado (UUID v7)
	CreatedAt string `json:"created_at"` // Timestamp no formato RFC3339
}

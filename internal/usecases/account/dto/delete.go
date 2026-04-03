package dto

// =============================================================================
// Delete Account DTOs
// =============================================================================

// DeleteInput representa os dados de entrada para deletar uma account.
type DeleteInput struct {
	ID string `json:"id"` // UUID v7 da account
}

// DeleteOutput representa os dados de saída após deleção.
type DeleteOutput struct {
	ID        string `json:"id"`
	DeletedAt string `json:"deleted_at"` // Timestamp da deleção
}

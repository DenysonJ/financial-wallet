package dto

// =============================================================================
// List Account DTOs
// =============================================================================

// ListInput representa os dados de entrada para listar accounts.
type ListInput struct {
	UserID     string `form:"-"`                       // Vem do JWT context
	Page       int    `form:"page"`                    // Página atual (1-indexed)
	Limit      int    `form:"limit"`                   // Itens por página
	Name       string `form:"name"  binding:"max=255"` // Filtro por nome
	Type       string `form:"type"  binding:"max=50"`  // Filtro por tipo
	ActiveOnly bool   `form:"active_only"`             // Apenas ativos
}

// ListOutput representa os dados de saída da listagem.
type ListOutput struct {
	Data       []GetOutput      `json:"data"`
	Pagination PaginationOutput `json:"pagination"`
}

// PaginationOutput representa os dados de paginação.
type PaginationOutput struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

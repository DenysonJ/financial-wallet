package dto

// =============================================================================
// List Role DTOs
// =============================================================================

// ListInput representa os dados de entrada para listar roles.
type ListInput struct {
	Page  int    `form:"page"`                   // Pagina atual (1-indexed)
	Limit int    `form:"limit"`                  // Itens por pagina
	Name  string `form:"name" binding:"max=100"` // Filtro por nome
}

// ListOutput representa os dados de saida da listagem.
type ListOutput struct {
	Data       []RoleOutput     `json:"data"`
	Pagination PaginationOutput `json:"pagination"`
}

// RoleOutput representa os dados de saida de uma role individual.
type RoleOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// PaginationOutput representa os dados de paginacao.
type PaginationOutput struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

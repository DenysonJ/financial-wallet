package dto

// =============================================================================
// Role Assignment DTOs
// =============================================================================

// AssignRoleInput representa os dados de entrada para atribuir uma role a um usuário.
type AssignRoleInput struct {
	UserID string `json:"user_id" binding:"required"` // ID do usuário
	RoleID string `json:"-"`                          // ID da role (from path param)
}

// RevokeRoleInput representa os dados de entrada para revogar uma role de um usuário.
type RevokeRoleInput struct {
	UserID string `json:"user_id" binding:"required"` // ID do usuário
	RoleID string `json:"-"`                          // ID da role (from path param)
}
